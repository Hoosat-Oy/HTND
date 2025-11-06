package client

import (
	"context"
	"time"

	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/server"
	"github.com/pkg/errors"

	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// Connect connects to the htnwalletd server with proper connection state handling
func Connect(address string) (pb.HtnwalletdClient, func(), error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(server.MaxDaemonSendMsgSize)),
	)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	// Give the connection up to 10 seconds to become ready
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Wait until the connection is ready (or fails)
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if !conn.WaitForStateChange(ctx, state) {
			// Context deadline exceeded or canceled
			conn.Close()
			return nil, nil, errors.New("failed to connect to htnwallet daemon: timeout after 10s - is it running? Run `htnwallet start-daemon`")
		}
	}

	client := pb.NewHtnwalletdClient(conn)
	closer := func() { conn.Close() }

	return client, closer, nil
}
