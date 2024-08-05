package server

import (
	"context"
	"strings"
	"time"

	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxRetries = 3
	retryDelay = 2 * time.Second
)

func (s *server) Send(_ context.Context, request *pb.SendRequest) (*pb.SendResponse, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		response, err := s.attemptSend(request)
		if err == nil {
			return response, nil
		}

		lastErr = err
		if shouldRetry(err) {
			time.Sleep(retryDelay)
			continue
		}

		return nil, err
	}

	return nil, lastErr
}

func (s *server) attemptSend(request *pb.SendRequest) (*pb.SendResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	unsignedTransactions, err := s.createUnsignedTransactions(request.ToAddress, request.Amount, request.IsSendAll,
		request.From, request.UseExistingChangeAddress)
	if err != nil {
		return nil, err
	}

	signedTransactions, err := s.signTransactions(unsignedTransactions, request.Password)
	if err != nil {
		return nil, err
	}

	txIDs, err := s.broadcast(signedTransactions, false)
	if err != nil {
		return nil, err
	}

	return &pb.SendResponse{TxIDs: txIDs, SignedTransactions: signedTransactions}, nil
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	if st.Code() != codes.Unknown {
		return false
	}

	errMsg := st.Message()

	fundsNotFound := strings.Contains(errMsg, "couldn't find funds to spend")
	alreadySpent := strings.Contains(errMsg, "error submitting transaction: Rejected transaction") && strings.Contains(errMsg, "already spent by transaction")

	return  fundsNotFound || alreadySpent
}