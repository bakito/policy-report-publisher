package report

import (
	"context"

	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	clientset "github.com/kyverno/kyverno/pkg/clients/kube"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler interface {
	Update(ctx context.Context, item *Item) error
	PolicyReportAvailable() (bool, error)
	RunAsLeader(ctx context.Context, cancel context.CancelFunc, leaseLockNamespace string, run func(ctx context.Context, handler Handler, cancel context.CancelFunc)) error
}

type handler struct {
	client     client.Client
	discovery  *discovery.DiscoveryClient
	logReports bool
	clientset  clientset.Interface
	counter    *prometheus.CounterVec
}

type Item struct {
	client.ObjectKey
	handlerID string
	result    prv1alpha2.PolicyReportResult
	source    interface{}
}

func ItemFor(handlerID string, namespace string, name string, result prv1alpha2.PolicyReportResult, source interface{}) *Item {
	return &Item{
		handlerID: handlerID,
		ObjectKey: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		result: result,
		source: source,
	}
}
