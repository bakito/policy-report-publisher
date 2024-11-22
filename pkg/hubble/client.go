package main

// https://github.com/cilium/cilium/tree/main/hubble/cmd/observe

import (
	"context"
	"errors"
	"io"

	observerpb "github.com/cilium/cilium/api/v1/observer"
	"github.com/cilium/cilium/hubble/cmd/common/conn"
	"github.com/cilium/cilium/hubble/pkg/logger"
	"github.com/cilium/cilium/pkg/hubble/relay/defaults"
	"github.com/cilium/cilium/pkg/inctimer"
	"github.com/cilium/cilium/pkg/time"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {

}

func a(ctx context.Context, hubbleServer string, timeout time.Duration) error {

	client, cleanup, err := hubbleClient(ctx, hubbleServer, timeout)
	if err != nil {
		return err
	}

	defer cleanup()

	flows := make(chan *observerpb.GetFlowsResponse, defaults.SortBufferMaxLen)

	retrieveFlowsFromPeer(ctx, client, nil, flows)

	go func() {
		updateTimer, updateTimerDone := inctimer.New()
		defer updateTimerDone()
		for {
			select {
			case <-updateTimer.After(s.opts.peerUpdateInterval):
				peers := s.peers.List()
				_, _ = fc.collect(gctx, g, peers, flows)
			case <-gctx.Done():
				return
			}
		}
	}()

	return nil
}

func hubbleClient(ctx context.Context, hubbleServer string, timeout time.Duration) (observerpb.ObserverClient, func() error, error) {
	// read flows from a hubble server
	hubbleConn, err := conn.New(ctx, hubbleServer, timeout)
	if err != nil {
		return nil, nil, err
	}
	logger.Logger.Debug("connected to Hubble API", "server", hubbleServer)
	cleanup := hubbleConn.Close
	client := observerpb.NewObserverClient(hubbleConn)

	return client, cleanup, nil
}

func retrieveFlowsFromPeer(
	ctx context.Context,
	client observerpb.ObserverClient,
	req *observerpb.GetFlowsRequest,
	flows chan<- *observerpb.GetFlowsResponse,
) error {
	c, err := client.GetFlows(ctx, req)
	if err != nil {
		return err
	}
	for {
		flow, err := c.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			if status.Code(err) == codes.Canceled {
				return nil
			}
			return err
		}

		select {
		case flows <- flow:
		case <-ctx.Done():
			return nil
		}
	}
}
