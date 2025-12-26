package report

import (
	"context"

	"github.com/bakito/policy-report-publisher-shared/types"
	clientset "github.com/kyverno/kyverno/pkg/clients/kube"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler interface {
	Update(ctx context.Context, item *types.Item) error
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
