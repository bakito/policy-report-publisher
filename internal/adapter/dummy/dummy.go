package dummy

import (
	"context"
	"time"

	"github.com/bakito/policy-report-publisher/pkg/api"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
)

func Run(_ context.Context, reportChan chan *api.Item) error {
	item := api.ItemFor("dummy", "ns", "pod", prv1alpha2.PolicyReportResult{Category: "dummy"}, "dummy")
	reportChan <- item
	time.Sleep(1 * time.Second)
	return nil
}
