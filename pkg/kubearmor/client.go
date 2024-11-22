package kubearmor

// https://github.com/kubearmor/kubearmor-client/blob/main/cmd/log.go

import (
	"fmt"
	"github.com/kubearmor/kubearmor-client/k8s"
	"github.com/kubearmor/kubearmor-client/log"
	klog "github.com/kubearmor/kubearmor-client/log"
	"github.com/kubearmor/kubearmor-client/profile"
	"github.com/kubearmor/kubearmor-client/utils"
	"os"
	"strconv"
)

const envKubearmorSvc = "KUBEARMOR_SERVICE"

var (
	port        int64 = 32767
	matchLabels       = map[string]string{"kubearmor-app": "kubearmor-relay"}
)

func a() {

	eventChan := make(chan klog.EventInfo)
	o := log.Options{
		EventChan: eventChan,
	}
	cl, _ := NewLogClient(o)
	go cl.WatchAlerts(o)

	// Consume events from the channel
	for event := range eventChan {
		fmt.Println(string(event.Data))
	}

}

func k8sClient() (*k8s.Client, error) {
	return k8s.ConnectK8sClient()
}

func NewLogClient(o log.Options) (*log.Feeder, error) {

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

	return log.NewClient(gRPC, o, client.K8sClientset)
}
