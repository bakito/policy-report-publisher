package main

import (
	"context"

	"github.com/bakito/policy-reporter-plugin/pkg/hubble"
	"github.com/bakito/policy-reporter-plugin/pkg/kubearmor"
	"github.com/bakito/policy-reporter-plugin/pkg/report"
)

func main() {
	handler, err := report.NewHandler()
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()

	reportChan := make(chan *report.Item)

	go kubearmor.Run(ctx, reportChan)
	go hubble.Run(ctx, reportChan)

	for report := range reportChan {
		err := handler.Update(ctx, report)
		if err != nil {
			panic(err)
		}
	}
}
