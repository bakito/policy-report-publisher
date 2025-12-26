package types

// Status specifies state of a policy result
const (
	StatusPass  PolicyResult = "pass"
	StatusFail  PolicyResult = "fail"
	StatusWarn  PolicyResult = "warn"
	StatusError PolicyResult = "error"
	StatusSkip  PolicyResult = "skip"
)

// Severity specifies priority of a policy result
const (
	SeverityCritical PolicySeverity = "critical"
	SeverityHigh     PolicySeverity = "high"
	SeverityMedium   PolicySeverity = "medium"
	SeverityLow      PolicySeverity = "low"
	SeverityInfo     PolicySeverity = "info"
)

// +kubebuilder:validation:Enum=pass;fail;warn;error;skip

// PolicyResult has one of the following values:
//   - pass: indicates that the policy requirements are met
//   - fail: indicates that the policy requirements are not met
//   - warn: indicates that the policy requirements and not met, and the policy is not scored
//   - error: indicates that the policy could not be evaluated
//   - skip: indicates that the policy was not selected based on user inputs or applicability
type PolicyResult string

// +kubebuilder:validation:Enum=critical;high;low;medium;info

// PolicySeverity has one of the following values:
// - critical
// - high
// - low
// - medium
// - info
type PolicySeverity string

// PolicyReportResult provides the result for an individual policy
type PolicyReportResult struct {
	// Source is an identifier for the policy engine that manages this report
	// +optional
	Source string `json:"source"`

	// Policy is the name or identifier of the policy
	Policy string `json:"policy"`

	// Rule is the name or identifier of the rule within the policy
	// +optional
	Rule string `json:"rule,omitempty"`

	Resources []NamespacedName `json:"resources,omitempty"`

	// Description is a short user friendly message for the policy rule
	Message string `json:"message,omitempty"`

	// Result indicates the outcome of the policy rule execution
	Result PolicyResult `json:"result,omitempty"`

	// Scored indicates if this result is scored
	Scored bool `json:"scored,omitempty"`

	// Properties provides additional information for the policy rule
	Properties map[string]string `json:"properties,omitempty"`

	// Timestamp indicates the time the result was found in nanos
	Timestamp int32 `json:"timestamp,omitempty"`

	// Category indicates policy category
	// +optional
	Category string `json:"category,omitempty"`

	// Severity indicates policy check result criticality
	// +optional
	Severity PolicySeverity `json:"severity,omitempty"`
}
