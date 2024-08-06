package router

import (
	"sync"
	"time"

	"github.com/Hoosat-Oy/HTND/app/protocol/protocolerrors"

	"github.com/Hoosat-Oy/HTND/app/appmessage"
	"github.com/pkg/errors"
)

var (
	// ErrTimeout signifies that one of the router functions had a timeout.
	ErrTimeout = protocolerrors.New(false, "timeout expired")

	// ErrRouteClosed indicates that a route was closed while reading/writing.
	ErrRouteClosed = errors.New("route is closed")
)

// Route represents an incoming or outgoing Router route
type Route struct {
	name    string
	channel chan appmessage.Message
	// closed and closeLock are used to protect us from writing to a closed channel
	// reads use the channel's built-in mechanism to check if the channel is closed
	closed    bool
	closeLock sync.Mutex
	capacity  int
}

// NewRoute create a new Route
func NewRoute(name string) *Route {
	return &Route{
		name:    name,
		channel: make(chan appmessage.Message),
		closed:  false,
	}
}

// Enqueue enqueues a message to the Route
func (r *Route) Enqueue(message appmessage.Message) error {
	r.closeLock.Lock()
	defer r.closeLock.Unlock()
	if r.closed {
		return errors.WithStack(ErrRouteClosed)
	}
	r.channel <- message
	log.Infof("Message enqueued, currently %d", len(r.channel))
	return nil
}

// MaybeEnqueue enqueues a message to the route,
// but doesn't throw an error if it's closed.
func (r *Route) MaybeEnqueue(message appmessage.Message) error {
	err := r.Enqueue(message)
	if errors.Is(err, ErrRouteClosed) {
		log.Infof("Couldn't send message to closed route '%s'", r.name)
		return nil
	}

	return err
}

// Dequeue dequeues a message from the Route
func (r *Route) Dequeue() (appmessage.Message, error) {
	message, isOpen := <-r.channel
	log.Infof("Message dequeued, currently %d", len(r.channel))
	if !isOpen {
		//log.Infof("Couldn't read message from closed route '%s'", r.name)
		return nil, errors.Wrapf(ErrRouteClosed, "route '%s' is closed", r.name)
	}
	return message, nil
}

// DequeueWithTimeout attempts to dequeue a message from the Route
// and returns an error if the given timeout expires first.
func (r *Route) DequeueWithTimeout(timeout time.Duration) (appmessage.Message, error) {
	select {
	case <-time.After(timeout):
		return nil, errors.Wrapf(ErrTimeout, "route '%s' got timeout after %s", r.name, timeout)
	case message, isOpen := <-r.channel:
		if !isOpen {
			return nil, errors.WithStack(ErrRouteClosed)
		}
		return message, nil
	}
}

// Close closes this route
func (r *Route) Close() {
	r.closeLock.Lock()
	defer r.closeLock.Unlock()

	r.closed = true
	close(r.channel)
}
