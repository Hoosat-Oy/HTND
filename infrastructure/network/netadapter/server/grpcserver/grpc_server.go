package grpcserver

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/server"
	"github.com/Hoosat-Oy/HTND/util/panics"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type gRPCServer struct {
	onConnectedHandler server.OnConnectedHandler
	listeningAddresses []string
	server             *grpc.Server
	name               string

	maxInboundConnections      int
	inboundConnectionCount     int
	inboundConnectionCountLock *sync.Mutex
}

// newGRPCServer creates a gRPC server
func newGRPCServer(listeningAddresses []string, maxMessageSize int, maxInboundConnections int, name string) *gRPCServer {
	log.Debugf("Created new %s GRPC server with maxMessageSize %d and maxInboundConnections %d", name, maxMessageSize, maxInboundConnections)
	return &gRPCServer{
		server:                     grpc.NewServer(grpc.MaxRecvMsgSize(maxMessageSize), grpc.MaxSendMsgSize(maxMessageSize)),
		listeningAddresses:         listeningAddresses,
		name:                       name,
		maxInboundConnections:      maxInboundConnections,
		inboundConnectionCount:     0,
		inboundConnectionCountLock: &sync.Mutex{},
	}
}

func (s *gRPCServer) Start() error {
	if s.onConnectedHandler == nil {
		return errors.New("onConnectedHandler is nil")
	}

	for _, listenAddress := range s.listeningAddresses {
		err := s.listenOn(listenAddress)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *gRPCServer) listenOn(listenAddr string) error {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return errors.Wrapf(err, "%s error listening on %s", s.name, listenAddr)
	}

	spawn(fmt.Sprintf("%s.gRPCServer.listenOn-Serve", s.name), func() {
		err := s.server.Serve(listener)
		if err != nil {
			panics.Exit(log, fmt.Sprintf("error serving %s on %s: %+v", s.name, listenAddr, err))
		}
	})

	log.Infof("%s Server listening on %s", s.name, listener.Addr())
	return nil
}

func (s *gRPCServer) Stop() error {
	const stopTimeout = 2 * time.Second

	stopChan := make(chan interface{})
	spawn("gRPCServer.Stop", func() {
		s.server.GracefulStop()
		close(stopChan)
	})

	select {
	case <-stopChan:
	case <-time.After(stopTimeout):
		log.Warnf("Could not gracefully stop %s: timed out after %s", s.name, stopTimeout)
		s.server.Stop()
	}
	return nil
}

// SetOnConnectedHandler sets the peer connected handler
// function for the server
func (s *gRPCServer) SetOnConnectedHandler(onConnectedHandler server.OnConnectedHandler) {
	s.onConnectedHandler = onConnectedHandler
}

func (s *gRPCServer) handleInboundConnection(ctx context.Context, stream grpcStream) error {
	connectionCount, err := s.incrementInboundConnectionCountAndLimitIfRequired()
	if err != nil {
		return err
	}
	defer s.decrementInboundConnectionCount()

	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return errors.Errorf("Error getting stream peer info from context")
	}
	tcpAddress, ok := peerInfo.Addr.(*net.TCPAddr)
	if !ok {
		return errors.Errorf("non-tcp connections are not supported")
	}

	connection := newConnection(s, tcpAddress, stream, nil)

	err = s.onConnectedHandler(connection)
	if err != nil {
		return err
	}

	log.Debugf("%s Incoming connection from %s #%d", s.name, peerInfo.Addr, connectionCount)

	<-connection.stopChan
	return nil
}

func (s *gRPCServer) incrementInboundConnectionCountAndLimitIfRequired() (int, error) {
	s.inboundConnectionCountLock.Lock()
	defer s.inboundConnectionCountLock.Unlock()

	if s.maxInboundConnections > 0 && s.inboundConnectionCount == s.maxInboundConnections {
		log.Warnf("Limit of %d %s inbound connections has been exceeded", s.maxInboundConnections, s.name)
		return s.inboundConnectionCount, errors.Errorf("limit of %d %s inbound connections has been exceeded", s.maxInboundConnections, s.name)
	}

	s.inboundConnectionCount++
	return s.inboundConnectionCount, nil
}

func (s *gRPCServer) decrementInboundConnectionCount() {
	s.inboundConnectionCountLock.Lock()
	defer s.inboundConnectionCountLock.Unlock()

	s.inboundConnectionCount--
}
