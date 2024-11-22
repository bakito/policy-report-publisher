package main

import (
	"context"

	"github.com/bakito/policy-reporter-plugin/pkg/hubble"
	"github.com/bakito/policy-reporter-plugin/pkg/kubearmor"
	"github.com/bakito/policy-reporter-plugin/pkg/report"
)

func main() {
	err, kc := report.NewKubeClient()
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()

	kubearmor.Run(ctx, kc)
	hubble.Run(ctx, kc)

}
