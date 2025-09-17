package api

import (
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PropertyCreated = "created"
	PropertyUpdated = "updated"
)

type Item struct {
	client.ObjectKey
	HandlerID string
	Result    prv1alpha2.PolicyReportResult
	Source    any
}

func ItemFor(handlerID string, namespace string, name string, result prv1alpha2.PolicyReportResult, source any) *Item {
	return &Item{
		HandlerID: handlerID,
		ObjectKey: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		Result: result,
		Source: source,
	}
}
