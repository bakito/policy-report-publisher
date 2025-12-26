package kubearmor

import (
	"time"

	"github.com/bakito/policy-report-publisher-shared/types"
)

const reportSource = "KubeArmor"

type Alert struct {
	Timestamp     int32     `json:"Timestamp"`
	UpdatedTime   time.Time `json:"UpdatedTime"`
	ClusterName   string    `json:"ClusterName"`
	HostName      string    `json:"HostName"`
	NamespaceName string    `json:"NamespaceName"`
	Owner         struct {
		Ref       string `json:"Ref"`
		Name      string `json:"Name"`
		Namespace string `json:"Namespace"`
	} `json:"Owner"`
	PodName           string `json:"PodName"`
	Labels            string `json:"Labels"`
	ContainerID       string `json:"ContainerID"`
	ContainerName     string `json:"ContainerName"`
	ContainerImage    string `json:"ContainerImage"`
	HostPPID          int    `json:"HostPPID"`
	HostPID           int    `json:"HostPID"`
	PPID              int    `json:"PPID"`
	PID               int    `json:"PID"`
	UID               int    `json:"UID"`
	ParentProcessName string `json:"ParentProcessName"`
	ProcessName       string `json:"ProcessName"`
	PolicyName        string `json:"PolicyName"`
	Severity          string `json:"Severity"`
	Type              string `json:"Type"`
	Source            string `json:"Source"`
	Operation         string `json:"Operation"`
	Resource          string `json:"Resource"`
	Data              string `json:"Data"`
	Enforcer          string `json:"Enforcer"`
	Action            string `json:"Action"`
	Result            string `json:"Result"`
	Cwd               string `json:"Cwd"`
}

func (a Alert) toItem() *types.Item {
	return types.ItemFor("kubearmor", a.NamespaceName, a.PodName, types.PolicyReportResult{
		Category: a.Type,
		Message:  a.Result,

		Severity: a.resultSeverity(),
		Policy:   a.PolicyName,
		// PolicyResult has one of the following values:
		//   - pass: indicates that the policy requirements are met
		//   - fail: indicates that the policy requirements are not met
		//   - warn: indicates that the policy requirements and not met, and the policy is not scored
		//   - error: indicates that the policy could not be evaluated
		//   - skip: indicates that the policy was not selected based on user inputs or applicability
		Result:    "fail",
		Scored:    true,
		Source:    reportSource,
		Timestamp: a.Timestamp,
		Properties: map[string]string{
			types.PropertyCreated: a.UpdatedTimeRFC3339(),
			types.PropertyUpdated: a.UpdatedTimeRFC3339(),
			"process-name":        a.ProcessName,
			"parent-process-name": a.ParentProcessName,
			"source":              a.Source,
			"operation":           a.Operation,
			"resource":            a.Resource,
			"cwd":                 a.Cwd,
		},
	}, &a)
}

func (a Alert) resultSeverity() types.PolicySeverity {
	// AubeArmor: severity: [1-10]  # --> optional (1 by default)

	// PolicySeverity has one of the following values:
	// - critical
	// - high
	// - low
	// - medium
	// - info

	switch a.Severity {
	case "1", "2":
		return "info"
	case "3", "4":
		return "medium"
	case "5", "6":
		return "low"
	case "7", "8":
		return "high"
	case "9", "10":
		return "critical"
	default:
		return "info"
	}
}

func (a Alert) UpdatedTimeRFC3339() string {
	return a.UpdatedTime.Format(time.RFC3339)
}
