package main

// https://github.com/kubearmor/kubearmor-client/blob/main/cmd/log.go

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kubearmor/kubearmor-client/k8s"
	"github.com/kubearmor/kubearmor-client/log"
	klog "github.com/kubearmor/kubearmor-client/log"
	"github.com/kubearmor/kubearmor-client/utils"
	"github.com/kyverno/policy-reporter-plugins/sdk/api"
)

const envKubearmorSvc = "KUBEARMOR_SERVICE"

var (
	port        int64 = 32767
	matchLabels       = map[string]string{"kubearmor-app": "kubearmor-relay"}
)

func main() {

	eventChan := make(chan klog.EventInfo)
	o := log.Options{
		EventChan: eventChan,
		LogFilter: "all",
	}
	cl, _ := NewLogClient(o)
	go cl.WatchAlerts(o)

	// Consume events from the channel
	for event := range eventChan {
		a := &Alert{}
		json.Unmarshal(event.Data, a)
		fmt.Println(a.Action)

		p := &api.Policy{
			Category:    v.Category,
			Name:        v.ID,
			Title:       v.Title,
			Description: v.Description,
			Severity:    v.Severity,
			Details:     make([]api.DetailsItem, 0),
			References:  make([]api.Reference, 0, len(v.References)),
			Engine: &api.Engine{
				Name:     "Trivy",
				Subjects: []string{"Pod", "ReplicaSet"},
			},
		}
		println(p.Name)
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

type Alert struct {
	Timestamp     int       `json:"Timestamp"`
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
