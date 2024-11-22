package main

// https://github.com/cilium/cilium/tree/main/hubble/cmd/observe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cilium/cilium/api/v1/flow"
	observerpb "github.com/cilium/cilium/api/v1/observer"
	"github.com/cilium/cilium/hubble/cmd"
	"github.com/cilium/cilium/hubble/pkg/defaults"
	"github.com/cilium/cilium/pkg/time"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func main() {
	if false {
		if err := cmd.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	ctx := context.TODO()

	client, cleanup, err := newClient(ctx, "localhost:4443")
	if err != nil {
		panic(err)
	}

	defer cleanup()

	req := &observerpb.GetFlowsRequest{
		Whitelist: []*flow.FlowFilter{
			{
				TrafficDirection: []flow.TrafficDirection{flow.TrafficDirection_EGRESS},
				Verdict:          []flow.Verdict{flow.Verdict_DROPPED},
			},
		},
	}

	err = getFlows(ctx, client, req)
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

func getFlows(ctx context.Context, client observerpb.ObserverClient, req *observerpb.GetFlowsRequest) error {
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
			println(r)
		}
	}
}
