package env

import (
	"os"
	"strings"
)

const (
	LogReports       = "LOG_REPORTS"
	LeaderElectionNS = "LEADER_ELECTION_NAMESPACE"

	HubbleServiceName = "HUBBLE_SERVICE"
	HubbleInsecure    = "HUBBLE_INSECURE"

	KubeArmorServiceName = "KUBE_ARMOR_SERVICE"
)

func Active(env string) bool {
	if i, ok := os.LookupEnv(env); ok && strings.EqualFold(i, "true") {
		return true
	}
	return false
}

func Empty(env string) bool {
	return strings.TrimSpace(os.Getenv(env)) == ""
}
