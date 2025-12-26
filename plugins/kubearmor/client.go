package kubearmor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bakito/policy-report-publisher-shared/types"
	"github.com/kubearmor/kubearmor-client/k8s"
	klog "github.com/kubearmor/kubearmor-client/log"
)

const EnvKubeArmorServiceName = "KUBE_ARMOR_SERVICE"

func Run(ctx context.Context, reportChan chan *types.Item) error {
	eventChan := make(chan klog.EventInfo)
	o := klog.Options{
		EventChan: eventChan,
		LogFilter: "all",
	}
	cl, err := newLogClient(o)
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

func newLogClient(o klog.Options) (*klog.Feeder, error) {
	if gRPC, ok := os.LookupEnv(EnvKubeArmorServiceName); ok {
		client, err := k8s.ConnectK8sClient()
		if err != nil {
			return nil, err
		}
		return klog.NewClient(gRPC, o, client.K8sClientset)
	}

	return nil, fmt.Errorf("kubearmor service name variable must %q be set", EnvKubeArmorServiceName)
}
