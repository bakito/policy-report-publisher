package dummy

import (
	"context"
	"time"

	"github.com/bakito/policy-report-publisher/internal/report"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
)

func Run(_ context.Context, reportChan chan *report.Item) error {
	item := report.ItemFor("dummy", "ns", "pod", prv1alpha2.PolicyReportResult{}, "dummy")
	reportChan <- item
	time.Sleep(1 * time.Second)
	return nil
}
