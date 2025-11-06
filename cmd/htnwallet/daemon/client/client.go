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

// Connect connects to the htnwalletd server, and returns the client instance
func Connect(address string) (pb.HtnwalletdClient, func(), error) {
	// Connection is local, so 1 second timeout is sufficient
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(server.MaxDaemonSendMsgSize)))
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	state := conn.GetState()
	if state == connectivity.Connecting || state == connectivity.Idle {
		if !conn.WaitForStateChange(ctx, state) {
			conn.Close()
			return nil, nil, errors.New("htnwallet daemon is not running, start it with `htnwallet start-daemon`")
		}
	}

	if conn.GetState() != connectivity.Ready {
		conn.Close()
		return nil, nil, errors.New("htnwallet daemon is not running, start it with `htnwallet start-daemon`")
	}

	return pb.NewHtnwalletdClient(conn), func() {
		conn.Close()
	}, nil
}
