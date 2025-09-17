package dummy

import (
	"context"
	"time"

	"github.com/bakito/policy-report-publisher/pkg/api"
)

func Run(_ context.Context, reportChan chan *api.Item) error {
	item := api.ItemFor("dummy", "ns", "pod", api.ReportResult{Policy: "dummy"}, "dummy")
	reportChan <- item
	time.Sleep(1 * time.Second)
	return nil
}
