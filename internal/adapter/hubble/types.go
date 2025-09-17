package hubble

import (
	"fmt"
	"strings"
	"time"

	"github.com/bakito/policy-report-publisher/internal/report"
	"github.com/cilium/cilium/api/v1/flow"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const reportSource = "Blocked Egress"

var consideredLabels = map[string]bool{
	"jenkins/label":                 true,
	"maintainer.fenaco.com/company": true,
	"maintainer.fenaco.com/team":    true,
	"product.fenaco.com/name":       true,
}

func toItem(f *flow.Flow) *report.Item {
	dest, protocol := destination(f)
	if dest == "" {
		return nil
	}

	pr := prv1alpha2.PolicyReportResult{
		Category: f.TrafficDirection.String(),
		Message:  f.DropReasonDesc.String(),

		Severity: "high",
		Policy:   "Egress Network Policy",
		Rule:     dest,
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
			report.PropertyCreated: updatedTimeRFC3339(f),
			report.PropertyUpdated: updatedTimeRFC3339(f),
			"protocol":             protocol,
		},
	}

	addPodLabels(f, pr)

	return report.ItemFor("clilum-blocked-egress", f.Source.Namespace, f.Source.PodName, pr, f)
}

func addPodLabels(f *flow.Flow, pr prv1alpha2.PolicyReportResult) {
	for _, podLabel := range f.Source.Labels {
		for l := range consideredLabels {
			if strings.HasPrefix(podLabel, "k8s:"+l+"=") {
				pr.Properties[l] = strings.SplitN(podLabel, "=", 2)[1]
			}
		}
	}
}

func destination(f *flow.Flow) (string, string) {
	if f.L4 != nil {
		if f.L4.GetTCP() != nil {
			if len(f.DestinationNames) == 0 {
				if f.Destination != nil && f.Destination.Namespace != "" {
					return fmt.Sprintf("%s/%s:%d", f.Destination.Namespace, f.Destination.PodName, f.L4.GetTCP().DestinationPort), "TCP"
				} else if f.IP != nil {
					return fmt.Sprintf("%s:%d", f.IP.Destination, f.L4.GetTCP().DestinationPort), "TCP"
				}
			} else {
				return fmt.Sprintf("%s:%d", f.DestinationNames[0], f.L4.GetTCP().DestinationPort), "TCP"
			}
		} else if f.L4.GetICMPv4() != nil {
			return f.IP.Destination, "ping"
		}
	}
	return "", ""
}

func updatedTimeRFC3339(f *flow.Flow) string {
	return f.GetTime().AsTime().Format(time.RFC3339)
}
