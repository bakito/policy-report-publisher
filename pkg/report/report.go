package report

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/bakito/policy-report-publisher/pkg/env"
	"github.com/bakito/policy-report-publisher/version"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	clientset "github.com/kyverno/kyverno/pkg/clients/kube"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	propCount = "count"

	PropertyCreated = "created"
	PropertyUpdated = "updated"
)

var PolicyReport = metav1.TypeMeta{Kind: "PolicyReport", APIVersion: prv1alpha2.GroupVersion.String()}

// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=wgpolicyk8s.io,resources=policyreports,verbs=get;list;watch;create;update;patch

func NewHandler() (Handler, error) {
	kc, dcl, cs, err := initKubeClient()
	if err != nil {
		return nil, err
	}

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "processed_items",
			Namespace: strings.ReplaceAll(version.Name, "-", "_"),
			Help:      "The number of processed items by adapter",
		},
		[]string{"adapter"},
	)

	prometheus.MustRegister(counter)

	return &handler{
		client:     kc,
		discovery:  dcl,
		clientset:  cs,
		logReports: env.Active(env.LogReports),
		counter:    counter,
	}, nil
}

func initKubeClient() (client.Client, *discovery.DiscoveryClient, clientset.Interface, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(prv1alpha2.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	restClientGetter := genericclioptions.ConfigFlags{}
	rawKubeConfigLoader := restClientGetter.ToRawKubeConfigLoader()

	config, err := rawKubeConfigLoader.ClientConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	dcl, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	cs, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	cl, err := client.New(config, client.Options{Scheme: scheme})
	return cl, dcl, cs, err
}

func (h *handler) PolicyReportAvailable() (bool, error) {
	resources, err := h.discovery.ServerPreferredResources()
	if err != nil {
		return false, err
	}

	for _, res := range resources {
		if res.GroupVersion == PolicyReport.APIVersion {
			for _, r := range res.APIResources {
				if r.Kind == PolicyReport.Kind {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (h *handler) Update(ctx context.Context, report *Item) error {
	if report.Name == "" {
		return nil
	}
	if h.logReports {
		b, err := json.Marshal(report.source)
		if err == nil {
			println(string(b))
		}
	}

	pod := &corev1.Pod{}
	err := h.client.Get(ctx, report.ObjectKey, pod)
	if err != nil {
		return err
	}

	h.counter.WithLabelValues(report.handlerID).Inc()

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		pol, err := h.getPolicyReport(ctx, report, pod)
		if err != nil {
			return err
		}
		_, err = controllerutil.CreateOrUpdate(ctx, h.client, pol, func() error {
			addResult(pol, report.result)
			return nil
		})
		return err
	})
}

func (h *handler) getPolicyReport(ctx context.Context, report *Item, pod *corev1.Pod) (*prv1alpha2.PolicyReport, error) {
	podID := string(pod.GetUID())
	policyID := fmt.Sprintf("prp-%s", podID)

	pol := &prv1alpha2.PolicyReport{}
	err := h.client.Get(ctx, types.NamespacedName{Namespace: report.Namespace, Name: policyID}, pol)
	if err != nil {
		if errors.IsNotFound(err) {
			pol = &prv1alpha2.PolicyReport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: report.Namespace,
					Name:      policyID,
				},
			}

			_ = controllerutil.SetOwnerReference(pod, pol, h.client.Scheme())
			pol.Scope = &corev1.ObjectReference{
				Namespace:  pod.Namespace,
				Name:       pod.Name,
				Kind:       "Pod",
				UID:        pod.GetUID(),
				APIVersion: pod.APIVersion,
			}
		} else {
			return nil, err
		}
	}

	return pol, nil
}

func addResult(pol *prv1alpha2.PolicyReport, result prv1alpha2.PolicyReportResult) {
	found := false

	for i, res := range pol.Results {
		if res.Source == result.Source && res.Policy == result.Policy && res.Rule == result.Rule {
			result.Properties = mergeProperties(pol.Results[i], result)
			pol.Results[i] = result
			found = true
		}
	}

	if !found {
		pol.Results = append(pol.Results, result)
	}
	pol.Summary.Fail++
}

func mergeProperties(oldReport prv1alpha2.PolicyReportResult, newReport prv1alpha2.PolicyReportResult) map[string]string {
	oldProps := oldReport.Properties
	cnt, err := strconv.Atoi(oldProps[propCount])
	if err != nil {
		cnt = 0
	}
	cnt++

	created := oldProps[PropertyCreated]

	newProps := newReport.Properties
	maps.Copy(oldProps, newProps)
	oldProps[propCount] = strconv.Itoa(cnt)
	if created != "" {
		oldProps[PropertyCreated] = created
	}
	return oldProps
}
