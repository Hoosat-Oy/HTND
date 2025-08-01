package netadapter

import (
	"fmt"
	"sync/atomic"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	routerpkg "github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"

	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/id"
	"github.com/Hoosat-Oy/HTND/infrastructure/network/netadapter/server"
)

// NetConnection is a wrapper to a server connection for use by services external to NetAdapter
type NetConnection struct {
	connection            server.Connection
	id                    *id.ID
	router                *routerpkg.Router
	onDisconnectedHandler server.OnDisconnectedHandler
	isRouterClosed        uint32
	shouldBanCounter      uint32
	ErrorMessage          error
}

func newNetConnection(connection server.Connection, routerInitializer RouterInitializer, name string) *NetConnection {
	router := routerpkg.NewRouter(name)

	netConnection := &NetConnection{
		connection:       connection,
		router:           router,
		shouldBanCounter: 0,
		ErrorMessage:     nil,
	}

	netConnection.connection.SetOnDisconnectedHandler(func() {
		log.Debugf("Disconnected from %s", netConnection)
		// If the disconnection came because of a network error and not because of the application layer, we
		// need to close the router as well.
		if atomic.AddUint32(&netConnection.isRouterClosed, 1) == 1 {
			netConnection.router.Close()
		}
		netConnection.onDisconnectedHandler()
	})

	routerInitializer(router, netConnection)

	return netConnection
}

func (c *NetConnection) start() {
	if c.onDisconnectedHandler == nil {
		panic(errors.New("onDisconnectedHandler is nil"))
	}

	c.connection.Start(c.router)
}

func (c *NetConnection) String() string {
	return fmt.Sprintf("<%s: %s>", c.id, c.connection)
}

// ID returns the ID associated with this connection
func (c *NetConnection) ID() *id.ID {
	return c.id
}

// SetID sets the ID associated with this connection
func (c *NetConnection) SetID(peerID *id.ID) {
	c.id = peerID
}

// Address returns the address associated with this connection
func (c *NetConnection) Address() string {
	return c.connection.Address().String()
}

// IsOutbound returns whether the connection is outbound
func (c *NetConnection) IsOutbound() bool {
	return c.connection.IsOutbound()
}

// NetAddress returns the NetAddress associated with this connection
func (c *NetConnection) NetAddress() *appmessage.NetAddress {
	return appmessage.NewNetAddress(c.connection.Address())
}

func (c *NetConnection) setOnDisconnectedHandler(onDisconnectedHandler server.OnDisconnectedHandler) {
	c.onDisconnectedHandler = onDisconnectedHandler
}

// Disconnect disconnects the given connection
func (c *NetConnection) Disconnect() {
	if atomic.AddUint32(&c.isRouterClosed, 1) == 1 {
		c.router.Close()
	}
}

// SetOnInvalidMessageHandler sets the invalid message handler for this connection
func (c *NetConnection) SetOnInvalidMessageHandler(onInvalidMessageHandler server.OnInvalidMessageHandler) {
	c.connection.SetOnInvalidMessageHandler(onInvalidMessageHandler)
}

func (c *NetConnection) ShouldWeBan(banLimit uint32) bool {
	if c.shouldBanCounter > banLimit {
		return true
	}
	c.shouldBanCounter += 1
	return false
}
