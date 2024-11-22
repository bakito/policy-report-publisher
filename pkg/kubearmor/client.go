package kubearmor

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/bakito/policy-reporter-plugin/pkg/report"
	"github.com/kubearmor/kubearmor-client/k8s"
	"github.com/kubearmor/kubearmor-client/log"
	klog "github.com/kubearmor/kubearmor-client/log"
	"github.com/kubearmor/kubearmor-client/utils"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const envKubearmorSvc = "KUBEARMOR_SERVICE"

var (
	port        int64 = 32767
	matchLabels       = map[string]string{"kubearmor-app": "kubearmor-relay"}
)

func Run(ctx context.Context, kc client.Client) {

	eventChan := make(chan klog.EventInfo)
	o := log.Options{
		EventChan: eventChan,
		LogFilter: "all",
	}
	cl, _, _ := NewLogClient(o)
	go func() {
		if err := cl.WatchAlerts(o); err != nil {
			panic(err)
		}
	}()

	// Consume events from the channel
	for event := range eventChan {
		a := &Alert{}
		_ = json.Unmarshal(event.Data, a)

		err := report.Update(ctx, kc, a.NamespaceName, a.PodName, func(pol *prv1alpha2.PolicyReport) error {
			a.addResult(pol)
			pol.Summary.Fail++
			return nil
		})
		if err != nil {
			panic(err)
		}
	}
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
