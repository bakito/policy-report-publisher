package kubearmor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/bakito/policy-reporter-plugin/pkg/report"
	"github.com/kubearmor/kubearmor-client/k8s"
	"github.com/kubearmor/kubearmor-client/log"
	klog "github.com/kubearmor/kubearmor-client/log"
	"github.com/kubearmor/kubearmor-client/utils"
)

const envKubearmorSvc = "KUBEARMOR_SERVICE"

var (
	port        int64 = 32767
	matchLabels       = map[string]string{"kubearmor-app": "kubearmor-relay"}
)

func Run(ctx context.Context, reportChan chan *report.Item) error {

	eventChan := make(chan klog.EventInfo)
	o := log.Options{
		EventChan: eventChan,
		LogFilter: "all",
	}
	cl, err := NewLogClient(o)
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	go func() {
		if err := cl.WatchAlerts(o); err != nil {
			errChan <- err
		}
	}()

	for {
		select {
		case err := <-errChan:
			close(eventChan)
			return err
		case <-ctx.Done():
			close(eventChan)
			return nil
		case event := <-eventChan:
			a := &Alert{}
			if err := json.Unmarshal(event.Data, a); err != nil {
				return fmt.Errorf("error unmarshalling alert: %w", err)
			}

			reportChan <- a.toItem()
		}
	}
}

func NewLogClient(o klog.Options) (*klog.Feeder, error) {

	gRPC := ""

	targetSvc := "kubearmor-relay"

	client, err := k8s.ConnectK8sClient()

	if err != nil {
		return nil, err
	}

	if o.GRPC != "" {
		gRPC = o.GRPC
	} else if val, ok := os.LookupEnv(envKubearmorSvc); ok {
		gRPC = val
	} else {
		pf, err := utils.InitiatePortForward(client, port, port, matchLabels, targetSvc)
		if err != nil {
			return nil, err
		}
		gRPC = "localhost:" + strconv.FormatInt(pf.LocalPort, 10)
	}

	lc, err := log.NewClient(gRPC, o, client.K8sClientset)
	return lc, err
}
