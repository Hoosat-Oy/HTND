package server

import (
	"context"

	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/pb"
	"github.com/Hoosat-Oy/HTND/version"
)

func (s *server) GetVersion(_ context.Context, _ *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return &pb.GetVersionResponse{
		Version: version.Version(),
	}, nil
}
