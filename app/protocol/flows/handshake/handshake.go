package handshake

import (
	"sync/atomic"
	"time"

	"github.com/Hoosat-Oy/HTND/domain"

	"github.com/Hoosat-Oy/HTND/app/protocol/common"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/addressmanager"

	"github.com/Hoosat-Oy/HTND/infrastructure/config"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	peerpkg "github.com/Hoosat-Oy/HTND/app/protocol/peer"
	routerpkg "github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

// HandleHandshakeContext is the interface for the context needed for the HandleHandshake flow.
type HandleHandshakeContext interface {
	Config() *config.Config
	NetAdapter() *netadapter.NetAdapter
	Domain() domain.Domain
	AddressManager() *addressmanager.AddressManager
	AddToPeers(peer *peerpkg.Peer) error
	HasPeer(peer *peerpkg.Peer) bool
	RemoveFromPeers(peer *peerpkg.Peer)
	HandleError(err error, flowName string, isStopping *uint32, errChan chan<- error)
}

// HandleHandshake sets up the new_handshake protocol - It sends a version message and waits for an incoming
// version message, as well as a verack for the sent version
func HandleHandshake(context HandleHandshakeContext, netConnection *netadapter.NetConnection,
	receiveVersionRoute *routerpkg.Route, sendVersionRoute *routerpkg.Route, outgoingRoute *routerpkg.Route,
) (*peerpkg.Peer, error) {
	doneCount := int32(2)
	doneChan := make(chan struct{})
	isStopping := uint32(0)
	errChan := make(chan error, 1) // Buffered channel to avoid blocking

	peer := peerpkg.New(netConnection)
	var peerAddress *appmessage.NetAddress

	spawn("HandleHandshake-ReceiveVersion", func() {
		defer func() {
			if atomic.AddInt32(&doneCount, -1) == 0 {
				close(doneChan)
			}
		}()
		log.Debugf("Starting ReceiveVersion for peer %v", peer)
		// Pass a deadline to ReceiveVersion to enforce timeout
		address, err := ReceiveVersion(context, receiveVersionRoute, outgoingRoute, peer)
		if err != nil {
			log.Debugf("ReceiveVersion error for peer %v: %v", peer, err)
			handleError(err, "ReceiveVersion", &isStopping, errChan)
			return
		}
		peerAddress = address
		log.Debugf("ReceiveVersion completed for peer %v", peer)
	})

	spawn("HandleHandshake-SendVersion", func() {
		defer func() {
			if atomic.AddInt32(&doneCount, -1) == 0 {
				close(doneChan)
			}
		}()
		log.Debugf("Starting SendVersion for peer %v", peer)
		// Pass a deadline to SendVersion to enforce timeout
		err := SendVersion(context, sendVersionRoute, outgoingRoute, peer)
		if err != nil {
			log.Debugf("SendVersion error for peer %v: %v", peer, err)
			handleError(err, "SendVersion", &isStopping, errChan)
			return
		}
		log.Debugf("SendVersion completed for peer %v", peer)
	})

	select {
	case err := <-errChan:
		return nil, err
	case <-doneChan:
	case <-time.After(30 * time.Second):
		log.Warnf("Handshake timed out for peer %v after 30 seconds", peer)
		return nil, errors.Wrap(common.ErrHandshakeTimeout, "handshake failed due to timeout")
	}

	err := context.AddToPeers(peer)
	if err != nil {
		if errors.Is(err, common.ErrPeerWithSameIDExists) {
			return nil, errors.Wrap(err, "peer already exists")
		}
		return nil, err
	}

	if peerAddress != nil {
		err := context.AddressManager().AddAddresses(peerAddress)
		if err != nil {
			return nil, err
		}
	}
	log.Debugf("Handshake completed for peer %v", peer)
	return peer, nil
}

// Handshake is different from other flows, since in it should forward router.ErrRouteClosed to errChan
// Therefore we implement a separate handleError for new_handshake
func handleError(err error, flowName string, isStopping *uint32, errChan chan error) {
	if errors.Is(err, routerpkg.ErrRouteClosed) {
		if atomic.AddUint32(isStopping, 1) == 1 {
			errChan <- err
		}
		return
	}

	if protocolErr := (protocolerrors.ProtocolError{}); errors.As(err, &protocolErr) {
		log.Debugf("Handshake protocol error from %s: %s", flowName, err)
		if atomic.AddUint32(isStopping, 1) == 1 {
			errChan <- err
		}
		return
	}
	panic(err)
}
