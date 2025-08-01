package protocol

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Hoosat-Oy/HTND/app/protocol/common"
	"github.com/Hoosat-Oy/HTND/app/protocol/flows/ready"
	v6 "github.com/Hoosat-Oy/HTND/app/protocol/flows/v6"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/Hoosat-Oy/HTND/app/protocol/flows/handshake"
	peerpkg "github.com/Hoosat-Oy/HTND/app/protocol/peer"
	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/addressmanager"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/connmanager"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter"
	routerpkg "github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

func isLocalAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		fmt.Println("Error parsing address:", err)
		return false
	}

	if host == "localhost" {
		return true
	}

	loopbackIPs := []string{"127.0.0.1", "::1"}
	for _, ip := range loopbackIPs {
		if host == ip {
			return true
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Error getting local network interfaces:", err)
		return false
	}

	for _, addr := range addrs {
		var ip net.IP
		if ipNet, ok := addr.(*net.IPNet); ok {
			ip = ipNet.IP
		} else if ipAddr, ok := addr.(*net.IPAddr); ok {
			ip = ipAddr.IP
		}

		if ip != nil && ip.String() == host {
			return true
		}
	}

	return false
}

func (m *Manager) routerInitializer(router *routerpkg.Router, netConnection *netadapter.NetConnection) {
	// isStopping flag is raised the moment that the connection associated with this router is disconnected
	// errChan is used by the flow goroutines to return to runFlows when an error occurs.
	// They are both initialized here and passed to register flows.
	isStopping := uint32(0)
	errChan := make(chan error, 1)

	receiveVersionRoute, sendVersionRoute, receiveReadyRoute := registerHandshakeRoutes(router)

	// After flows were registered - spawn a new thread that will wait for connection to finish initializing
	// and start receiving messages
	spawn("routerInitializer-runFlows", func() {
		m.routersWaitGroup.Add(1)
		defer m.routersWaitGroup.Done()

		if atomic.LoadUint32(&m.isClosed) == 1 {
			panic(errors.Errorf("tried to initialize router when the protocol manager is closed"))
		}
		log.Debugf("initializing route for %s", netConnection)
		isBanned, err := m.context.ConnectionManager().IsBanned(netConnection)
		if err != nil && !errors.Is(err, addressmanager.ErrAddressNotFound) {
			m.handleError(err, netConnection, router.OutgoingRoute())
			return
		}
		if isBanned {
			log.Infof("Peer %s is banned. Disconnecting...", netConnection)
			netConnection.Disconnect()
			return
		}

		netConnection.SetOnInvalidMessageHandler(func(err error) {
			if atomic.AddUint32(&isStopping, 1) == 1 {
				errChan <- protocolerrors.Wrap(true, err, "received bad message")
			}
		})
		log.Debugf("Doing Handshake with %s", netConnection)
		peer, err := handshake.HandleHandshake(m.context, netConnection, receiveVersionRoute, sendVersionRoute, router.OutgoingRoute())
		if err != nil {
			// non-blocking read from channel
			select {
			case innerError := <-errChan:
				if errors.Is(err, routerpkg.ErrRouteClosed) {
					m.handleError(innerError, netConnection, router.OutgoingRoute())
				} else {
					log.Errorf("Peer %s sent invalid message: %s", netConnection, innerError)
					m.handleError(err, netConnection, router.OutgoingRoute())
				}
			default:
				m.handleError(err, netConnection, router.OutgoingRoute())
			}
			netConnection.ErrorMessage = err
			return
		}
		defer m.context.RemoveFromPeers(peer)

		var flows []*common.Flow
		log.Debugf("Registering p2p flows for peer %s for protocol version %d", peer, peer.ProtocolVersion())
		switch peer.ProtocolVersion() {
		case 6:
			flows = v6.Register(m, netConnection, router, errChan, &isStopping)
		default:
			panic(errors.Errorf("no way to handle protocol version %d", peer.ProtocolVersion()))
		}
		log.Debugf("Registered p2p flows for peer %s.", peer)

		err = ready.HandleReady(receiveReadyRoute, router.OutgoingRoute(), peer)
		if err != nil {
			m.handleError(err, netConnection, router.OutgoingRoute())
			return
		}
		log.Debugf("HandleReady for peer %s", peer)
		removeHandshakeRoutes(router)

		log.Infof("Registered p2p flows and running them for peer %s", peer)
		flowsWaitGroup := &sync.WaitGroup{}
		err = m.runFlows(flows, peer, errChan, flowsWaitGroup)
		if err != nil {
			m.handleError(err, netConnection, router.OutgoingRoute())
			// We call `flowsWaitGroup.Wait()` in two places instead of deferring, because
			// we already defer `m.routersWaitGroup.Done()`, so we try to avoid error prone
			// and confusing use of multiple dependent defers.
			flowsWaitGroup.Wait()
			return
		}
		flowsWaitGroup.Wait()
	})
}

func (m *Manager) handleError(err error, netConnection *netadapter.NetConnection, outgoingRoute *routerpkg.Route) {
	netConnection.ErrorMessage = err
	if protocolErr := (protocolerrors.ProtocolError{}); errors.As(err, &protocolErr) {
		if m.context.Config().EnableBanning && protocolErr.ShouldBan {
			log.Warnf("Banning %s (reason: %s)", netConnection, protocolErr.Cause)

			err := m.context.ConnectionManager().Ban(netConnection)
			if err != nil && !errors.Is(err, connmanager.ErrCannotBanPermanent) {
				panic(err)
			}

			err = outgoingRoute.Enqueue(appmessage.NewMsgReject(protocolErr.Error()))
			if err != nil && !errors.Is(err, routerpkg.ErrRouteClosed) {
				panic(err)
			}
		}
		log.Debugf("Disconnecting from %s (reason: %s)", netConnection, protocolErr.Cause)
		netConnection.Disconnect()
		return
	}

	if errors.Is(err, common.ErrHandshakeTimeout) {
		log.Debugf("Handshake timed out for connection %v. Disconnecting..", netConnection)
		netConnection.Disconnect()
		return
	}

	if errors.Is(err, common.ErrPeerWithSameIDExists) {
		log.Debugf("Peer already exists %v. Disconnecting..", netConnection)
		netConnection.Disconnect()
		return
	}
	if errors.Is(err, routerpkg.ErrTimeout) {
		log.Debugf("Got timeout from %s. Disconnecting...", netConnection)
		netConnection.Disconnect()
		return
	}
	if errors.Is(err, routerpkg.ErrRouteClosed) {
		return
	}
	panic(err)
}

// RegisterFlow registers a flow to the given router.
func (m *Manager) RegisterFlow(name string, router *routerpkg.Router, messageTypes []appmessage.MessageCommand, isStopping *uint32,
	errChan chan error, initializeFunc common.FlowInitializeFunc) *common.Flow {

	route, err := router.AddIncomingRoute(name, messageTypes)
	if err != nil {
		panic(err)
	}

	return m.registerFlowForRoute(route, name, isStopping, errChan, initializeFunc)
}

// RegisterFlowWithCapacity registers a flow to the given router with a custom capacity.
func (m *Manager) RegisterFlowWithCapacity(name string, capacity int, router *routerpkg.Router,
	messageTypes []appmessage.MessageCommand, isStopping *uint32,
	errChan chan error, initializeFunc common.FlowInitializeFunc) *common.Flow {

	route, err := router.AddIncomingRouteWithCapacity(name, capacity, messageTypes)
	if err != nil {
		panic(err)
	}

	return m.registerFlowForRoute(route, name, isStopping, errChan, initializeFunc)
}

func (m *Manager) registerFlowForRoute(route *routerpkg.Route, name string, isStopping *uint32,
	errChan chan error, initializeFunc common.FlowInitializeFunc) *common.Flow {

	return &common.Flow{
		Name: name,
		ExecuteFunc: func(peer *peerpkg.Peer) {
			err := initializeFunc(route, peer)
			if err != nil {
				m.context.HandleError(err, name, isStopping, errChan)
				return
			}
		},
	}
}

// RegisterOneTimeFlow registers a one-time flow (that exits once some operations are done) to the given router.
func (m *Manager) RegisterOneTimeFlow(name string, router *routerpkg.Router, messageTypes []appmessage.MessageCommand,
	isStopping *uint32, stopChan chan error, initializeFunc common.FlowInitializeFunc) *common.Flow {

	route, err := router.AddIncomingRoute(name, messageTypes)
	if err != nil {
		panic(err)
	}

	return &common.Flow{
		Name: name,
		ExecuteFunc: func(peer *peerpkg.Peer) {
			defer func() {
				err := router.RemoveRoute(messageTypes)
				if err != nil {
					panic(err)
				}
			}()

			err := initializeFunc(route, peer)
			if err != nil {
				m.context.HandleError(err, name, isStopping, stopChan)
				return
			}
		},
	}
}

func registerHandshakeRoutes(router *routerpkg.Router) (
	receiveVersionRoute, sendVersionRoute, receiveReadyRoute *routerpkg.Route) {
	receiveVersionRoute, err := router.AddIncomingRoute("recieveVersion - incoming", []appmessage.MessageCommand{appmessage.CmdVersion})
	if err != nil {
		panic(err)
	}

	sendVersionRoute, err = router.AddIncomingRoute("sendVersion - incoming", []appmessage.MessageCommand{appmessage.CmdVerAck})
	if err != nil {
		panic(err)
	}

	receiveReadyRoute, err = router.AddIncomingRoute("recieveReady - incoming", []appmessage.MessageCommand{appmessage.CmdReady})
	if err != nil {
		panic(err)
	}

	return receiveVersionRoute, sendVersionRoute, receiveReadyRoute
}

func removeHandshakeRoutes(router *routerpkg.Router) {
	err := router.RemoveRoute([]appmessage.MessageCommand{appmessage.CmdVersion, appmessage.CmdVerAck, appmessage.CmdReady})
	if err != nil {
		panic(err)
	}
}
