package env

import (
	"os"
	"strings"
)

const (
	LogReports       = "LOG_REPORTS"
	LeaderElectionNS = "LEADER_ELECTION_NAMESPACE"
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
