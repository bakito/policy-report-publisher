package report

import (
	"context"

	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler interface {
	Update(ctx context.Context, item *Item) error
	PolicyReportAvailable() (bool, error)
}

type handler struct {
	client    client.Client
	discovery *discovery.DiscoveryClient
}

type Item struct {
	client.ObjectKey
	result prv1alpha2.PolicyReportResult
}

func ItemFor(namespace string, name string, result prv1alpha2.PolicyReportResult) *Item {
	return &Item{
		ObjectKey: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		result: result,
	}
}
