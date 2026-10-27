package report

import (
	"context"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/discovery"
	kubernetes "k8s.io/client-go/kubernetes"
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
	clientset  kubernetes.Interface
	counter    *prometheus.CounterVec
}

type Item struct {
	client.ObjectKey
	handlerID string
	result    openreportsv1alpha1.ReportResult
	source    any
}

func ItemFor(handlerID string, namespace string, name string, result openreportsv1alpha1.ReportResult, source any) *Item {
	return &Item{
		handlerID: handlerID,
		Namespace: namespace,
		Name:      name,
		result:    result,
		source:    source,
	}
}
