package report

import (
	"context"

	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewHandler() (Handler, error) {
	client, err := initKubeClient()
	if err != nil {
		return nil, err
	}
	return &handler{
		client: client,
	}, nil
}

func initKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(prv1alpha2.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	restClientGetter := genericclioptions.ConfigFlags{}
	rawKubeConfigLoader := restClientGetter.ToRawKubeConfigLoader()

	config, err := rawKubeConfigLoader.ClientConfig()
	if err != nil {
		panic(err)
	}

	return client.New(config, client.Options{Scheme: scheme})
}

func (h *handler) Update(ctx context.Context, report *Item) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		pol, err := h.getPolicyReport(ctx, report)
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

func (h *handler) getPolicyReport(ctx context.Context, report *Item) (*prv1alpha2.PolicyReport, error) {

	pod := &corev1.Pod{}
	err := h.client.Get(ctx, report.ObjectKey, pod)
	if err != nil {
		return nil, nil
	}

	pol := &prv1alpha2.PolicyReport{}
	err = h.client.Get(ctx, types.NamespacedName{Namespace: pod.GetNamespace(), Name: string(pod.GetUID())}, pol)

	if err != nil {
		if errors.IsNotFound(err) {
			pol = &prv1alpha2.PolicyReport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: pod.GetNamespace(),
					Name:      string(pod.GetUID()),
				},
			}

			_ = controllerutil.SetOwnerReference(pod, pol, h.client.Scheme())
			pol.Scope = &corev1.ObjectReference{
				Namespace:  pod.Namespace,
				Name:       pod.Name,
				Kind:       pod.Kind,
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
		if res.Source == result.Source && res.Policy == result.Policy {
			pol.Results[i] = result
			found = true
		}
	}

	if !found {
		pol.Results = append(pol.Results, result)
	}
}
