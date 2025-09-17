package hubble

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bakito/policy-report-publisher/internal/env"
	"github.com/bakito/policy-report-publisher/internal/report"
	"github.com/cilium/cilium/api/v1/flow"
	observerpb "github.com/cilium/cilium/api/v1/observer"
	"github.com/cilium/cilium/hubble/pkg/defaults"
	"github.com/cilium/cilium/pkg/time"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func Run(ctx context.Context, reportChan chan *report.Item) error {
	client, cleanup, err := newClient()
	if err != nil {
		return err
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

	return getFlows(ctx, client, reportChan, req)
}

func newClient() (observerpb.ObserverClient, func() error, error) {
	var gRPC string
	if val, ok := os.LookupEnv(env.HubbleServiceName); ok {
		gRPC = val
	} else {
		return nil, nil, fmt.Errorf("hubble service name variable must %q be set", env.HubbleServiceName)
	}

	// read flows from a hubble server
	hubbleConn, err := newConn(gRPC)
	if err != nil {
		return nil, nil, err
	}
	cleanup := hubbleConn.Close
	client := observerpb.NewObserverClient(hubbleConn)

	return client, cleanup, err
}

// New creates a new gRPC client connection to the target.
func newConn(target string) (*grpc.ClientConn, error) {
	var creds credentials.TransportCredentials

	if env.Active(env.HubbleInsecure) {
		creds = insecure.NewCredentials()
	} else {
		tlsConfig := tls.Config{
			InsecureSkipVerify: true, // #nosec G402
		}
		creds = credentials.NewTLS(&tlsConfig)
	}
	t := strings.TrimPrefix(target, defaults.TargetTLSPrefix)
	conn, err := grpc.NewClient(t,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(12*time.Second)))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", target, err)
	}
	return conn, nil
}

func getFlows(ctx context.Context, client observerpb.ObserverClient, reportChan chan *report.Item, req *observerpb.GetFlowsRequest) error {
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
				item := toItem(r.Flow)
				if item != nil {
					reportChan <- item
				}
			}
		}
	}
}

func ignoreFlow(f *flow.Flow) bool {
	return f == nil || f.Source == nil || f.Source.PodName == "" ||
		f.L4 == nil || (f.L4.GetTCP() == nil && f.L4.GetICMPv4() == nil)
}
