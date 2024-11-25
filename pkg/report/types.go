package report

import (
	"context"

	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	clientset "github.com/kyverno/kyverno/pkg/clients/kube"
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
}

type Item struct {
	client.ObjectKey
	result prv1alpha2.PolicyReportResult
	source interface{}
}

func ItemFor(namespace string, name string, result prv1alpha2.PolicyReportResult, source interface{}) *Item {
	return &Item{
		ObjectKey: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		result: result,
		source: source,
	}
}
