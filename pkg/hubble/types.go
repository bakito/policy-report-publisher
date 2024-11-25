package hubble

import (
	"fmt"
	"time"

	"github.com/bakito/policy-report-publisher/pkg/report"
	"github.com/cilium/cilium/api/v1/flow"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const reportSource = "Cilium Hubble"

func toItem(f *flow.Flow) *report.Item {
	pr := prv1alpha2.PolicyReportResult{
		Category: f.TrafficDirection.String(),
		Message:  f.DropReasonDesc.String(),

		Severity: "high",
		Policy:   "Egress Network Policy",
		// PolicyResult has one of the following values:
		//   - pass: indicates that the policy requirements are met
		//   - fail: indicates that the policy requirements are not met
		//   - warn: indicates that the policy requirements and not met, and the policy is not scored
		//   - error: indicates that the policy could not be evaluated
		//   - skip: indicates that the policy was not selected based on user inputs or applicability
		Result: "fail",
		Scored: true,
		Source: reportSource,
		Timestamp: metav1.Timestamp{
			Nanos: f.Time.GetNanos(),
		},
		Properties: map[string]string{
			"UpdatedTime": f.GetTime().AsTime().Format(time.RFC3339),
		},
	}

	if f.L4 != nil {
		if f.L4.GetTCP() != nil {
			if len(f.DestinationNames) == 0 && f.Destination != nil {
				pr.Properties[f.Destination.Namespace+"/"+f.Destination.PodName] = fmt.Sprintf("%d", f.L4.GetTCP().DestinationPort)
			} else {
				for _, name := range f.DestinationNames {
					pr.Properties[name] = fmt.Sprintf("%d", f.L4.GetTCP().DestinationPort)
				}
			}
		} else if f.L4.GetICMPv4() != nil {
			pr.Properties["ping "+f.IP.Destination] =
				fmt.Sprintf("type: %d / code: %d", f.L4.GetICMPv4().Type, f.L4.GetICMPv4().Code)
		}
	}

	return report.ItemFor(f.Source.Namespace, f.Source.PodName, pr, f)
}
