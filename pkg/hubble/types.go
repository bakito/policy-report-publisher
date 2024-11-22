package hubble

import (
	"time"

	"github.com/bakito/policy-reporter-plugin/pkg/report"
	"github.com/cilium/cilium/api/v1/flow"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const reportSource = "Cilium Hubble"

func toItem(f *flow.Flow) *report.Item {
	return report.ItemFor(f.Source.Namespace, f.Source.PodName, prv1alpha2.PolicyReportResult{
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
	},
	)
}
