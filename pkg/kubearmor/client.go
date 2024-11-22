package main

// https://github.com/kubearmor/kubearmor-client/blob/main/cmd/log.go

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/kubearmor/kubearmor-client/k8s"
	"github.com/kubearmor/kubearmor-client/log"
	klog "github.com/kubearmor/kubearmor-client/log"
	"github.com/kubearmor/kubearmor-client/utils"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const envKubearmorSvc = "KUBEARMOR_SERVICE"

var (
	port        int64 = 32767
	matchLabels       = map[string]string{"kubearmor-app": "kubearmor-relay"}
)

func main() {

	_ = prv1alpha2.AddToScheme(scheme.Scheme)

	restClientGetter := genericclioptions.ConfigFlags{}
	rawKubeConfigLoader := restClientGetter.ToRawKubeConfigLoader()

	config, err := rawKubeConfigLoader.ClientConfig()
	if err != nil {
		panic(err)
	}

	kc, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()

	eventChan := make(chan klog.EventInfo)
	o := log.Options{
		EventChan: eventChan,
		LogFilter: "all",
	}
	cl, _, _ := NewLogClient(o)
	go cl.WatchAlerts(o)

	// Consume events from the channel
	for event := range eventChan {
		a := &Alert{}
		_ = json.Unmarshal(event.Data, a)

		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			pol, err := getPolicyReport(ctx, kc, a)
			if err != nil {
				return err
			}
			_, err = controllerutil.CreateOrUpdate(ctx, kc, pol, func() error {
				a.addResult(pol)
				pol.Summary.Fail++
				return nil
			})
			return err
		})
		if err != nil {
			panic(err)
		}
	}
}

func getPolicyReport(ctx context.Context, kc client.Client, alert *Alert) (*prv1alpha2.PolicyReport, error) {

	pod := &corev1.Pod{}
	err := kc.Get(ctx, types.NamespacedName{Namespace: alert.NamespaceName, Name: alert.PodName}, pod)
	if err != nil {
		return nil, nil
	}

	pol := &prv1alpha2.PolicyReport{}
	err = kc.Get(ctx, types.NamespacedName{Namespace: pod.GetNamespace(), Name: string(pod.GetUID())}, pol)

	if err != nil {
		if errors.IsNotFound(err) {
			pol = &prv1alpha2.PolicyReport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: pod.GetNamespace(),
					Name:      string(pod.GetUID()),
				},
			}

			_ = controllerutil.SetOwnerReference(pod, pol, scheme.Scheme)
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

func NewLogClient(o klog.Options) (*klog.Feeder, *k8s.Client, error) {

	gRPC := ""

	targetSvc := "kubearmor-relay"

	client, err := k8s.ConnectK8sClient()

	if err != nil {
		return nil, nil, err
	}

	if o.GRPC != "" {
		gRPC = o.GRPC
	} else if val, ok := os.LookupEnv(envKubearmorSvc); ok {
		gRPC = val
	} else {
		pf, err := utils.InitiatePortForward(client, port, port, matchLabels, targetSvc)
		if err != nil {
			return nil, nil, err
		}
		gRPC = "localhost:" + strconv.FormatInt(pf.LocalPort, 10)
	}

	lc, err := log.NewClient(gRPC, o, client.K8sClientset)
	return lc, client, err
}

type Alert struct {
	Timestamp     int32     `json:"Timestamp"`
	UpdatedTime   time.Time `json:"UpdatedTime"`
	ClusterName   string    `json:"ClusterName"`
	HostName      string    `json:"HostName"`
	NamespaceName string    `json:"NamespaceName"`
	Owner         struct {
		Ref       string `json:"Ref"`
		Name      string `json:"Name"`
		Namespace string `json:"Namespace"`
	} `json:"Owner"`
	PodName           string `json:"PodName"`
	Labels            string `json:"Labels"`
	ContainerID       string `json:"ContainerID"`
	ContainerName     string `json:"ContainerName"`
	ContainerImage    string `json:"ContainerImage"`
	HostPPID          int    `json:"HostPPID"`
	HostPID           int    `json:"HostPID"`
	PPID              int    `json:"PPID"`
	PID               int    `json:"PID"`
	UID               int    `json:"UID"`
	ParentProcessName string `json:"ParentProcessName"`
	ProcessName       string `json:"ProcessName"`
	PolicyName        string `json:"PolicyName"`
	Severity          string `json:"Severity"`
	Type              string `json:"Type"`
	Source            string `json:"Source"`
	Operation         string `json:"Operation"`
	Resource          string `json:"Resource"`
	Data              string `json:"Data"`
	Enforcer          string `json:"Enforcer"`
	Action            string `json:"Action"`
	Result            string `json:"Result"`
	Cwd               string `json:"Cwd"`
}

func (a Alert) addResult(pol *prv1alpha2.PolicyReport) {

	pr := prv1alpha2.PolicyReportResult{
		Category: a.Type,
		Message:  a.Result,

		Severity: a.resultSeverity(),
		Policy:   a.PolicyName,
		// PolicyResult has one of the following values:
		//   - pass: indicates that the policy requirements are met
		//   - fail: indicates that the policy requirements are not met
		//   - warn: indicates that the policy requirements and not met, and the policy is not scored
		//   - error: indicates that the policy could not be evaluated
		//   - skip: indicates that the policy was not selected based on user inputs or applicability
		Result: "fail",
		Scored: true,
		Source: "KubeArmor",
		Timestamp: metav1.Timestamp{
			Nanos: a.Timestamp,
		},
		Properties: map[string]string{
			"ProcessName":       a.ProcessName,
			"ParentProcessName": a.ParentProcessName,
			"Source":            a.Source,
			"Operation":         a.Operation,
			"Resource":          a.Resource,
			"Cwd":               a.Cwd,
			"UpdatedTime":       a.UpdatedTime.Format(time.RFC3339),
		},
	}

	found := false

	for i, res := range pol.Results {
		if res.Source == "KubeArmor" && res.Policy == a.PolicyName {
			pol.Results[i] = pr
			found = true
		}
	}

	if !found {
		pol.Results = append(pol.Results, pr)
	}
}

func (a Alert) resultSeverity() prv1alpha2.PolicySeverity {
	// AubeArmor: severity: [1-10]  # --> optional (1 by default)

	// PolicySeverity has one of the following values:
	// - critical
	// - high
	// - low
	// - medium
	// - info

	switch a.Severity {
	case "1", "2":
		return "info"
	case "3", "4":
		return "medium"
	case "5", "6":
		return "low"
	case "7", "8":
		return "high"
	case "9", "10":
		return "critical"
	default:
		return "info"
	}

}
