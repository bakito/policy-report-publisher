package hubble

// https://github.com/cilium/cilium/tree/main/hubble/cmd/observe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bakito/policy-reporter-plugin/pkg/report"
	"github.com/cilium/cilium/api/v1/flow"
	observerpb "github.com/cilium/cilium/api/v1/observer"
	"github.com/cilium/cilium/hubble/pkg/defaults"
	"github.com/cilium/cilium/pkg/time"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Run(ctx context.Context, kc client.Client) {

	client, cleanup, err := newClient(ctx, "localhost:4443")
	if err != nil {
		panic(err)
	}

	defer func() { _ = cleanup() }()

	req := &observerpb.GetFlowsRequest{
		Follow: true,
		Whitelist: []*flow.FlowFilter{
			{
				TrafficDirection: []flow.TrafficDirection{flow.TrafficDirection_EGRESS},
				Verdict:          []flow.Verdict{flow.Verdict_DROPPED},
			},
		},
	}

	err = getFlows(ctx, client, kc, req)
	if err != nil {
		panic(err)
	}
}

func newClient(ctx context.Context, hubbleServer string) (observerpb.ObserverClient, func() error, error) {

	// read flows from a hubble server
	hubbleConn, err := newConn(ctx, hubbleServer, 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	cleanup := hubbleConn.Close
	client := observerpb.NewObserverClient(hubbleConn)

	return client, cleanup, err

}

// New creates a new gRPC client connection to the target.
func newConn(ctx context.Context, target string, dialTimeout time.Duration) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	tlsConfig := tls.Config{
		InsecureSkipVerify: true, // #nosec G402
	}

	creds := credentials.NewTLS(&tlsConfig)

	t := strings.TrimPrefix(target, defaults.TargetTLSPrefix)
	conn, err := grpc.DialContext(dialCtx, t, grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithReturnConnectionError(),
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(12*time.Second)))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", target, err)
	}
	return conn, nil
}

func getFlows(ctx context.Context, client observerpb.ObserverClient, kc client.Client, req *observerpb.GetFlowsRequest) error {
	b, err := client.GetFlows(ctx, req)
	if err != nil {
		return err
	}

	for {
		resp, err := b.Recv()
		switch {
		case errors.Is(err, io.EOF), errors.Is(err, context.Canceled):
			return nil
		case err == nil:
		default:
			if status.Code(err) == codes.Canceled {
				return nil
			}
			return err
		}

		switch r := resp.GetResponseTypes().(type) {
		case *observerpb.GetFlowsResponse_Flow:
			if !ignoreFlow(r.Flow) {
				err = report.Update(ctx, kc, r.Flow.Source.Namespace,
					r.Flow.Source.PodName,
					func(pol *prv1alpha2.PolicyReport) error {
						addResultFor(pol, r.Flow)
						pol.Summary.Fail++
						return nil
					},
				)
			}
			if err != nil {
				return err
			}
		}
	}
}

func ignoreFlow(f *flow.Flow) bool {
	return f != nil && f.L4 != nil && (f.L4.GetTCP() != nil || f.L4.GetICMPv4() != nil)
}
