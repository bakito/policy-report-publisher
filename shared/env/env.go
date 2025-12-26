package env

import (
	"os"
	"strings"
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
