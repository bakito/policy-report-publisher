package api

import (
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
	Result    ReportResult
	Source    any
}

func ItemFor(handlerID string, namespace string, name string, result ReportResult, source any) *Item {
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

type ReportResult struct {
	Source     string
	Policy     string
	Rule       string
	Properties map[string]string
}
