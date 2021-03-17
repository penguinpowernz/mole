package tunnel

import (
	"context"

	"github.com/AlexanderGrom/go-event"
)

// DefaultPool is setup so that you don't always need to provide a pool
var DefaultPool = FakePool{}

// Pool is a connection pool that allows the Tunnel to get it's
// client from the pool (if the connection already exists).  This
// saves RAM use as each SSH connection uses about 4MB of RAM.
type Pool interface {
	GetClient(string, string) (*Client, error)
}

// FakePool does not pool at all it simply creates a new client for
// every single call
type FakePool struct {
}

// GetClient will create a new client everytime it is called because
// this is a fake pool
func (pl FakePool) GetClient(addr, key string) (*Client, error) {
	cl, err := NewClient(addr, key)
	if err != nil {
		return nil, err
	}

	err = cl.Connect()
	return cl, err
}

// NewConnPool will return a new connection pool that stores active client connections
// or add and connects to clients that aren't already in the pool
func NewConnPool(ctx context.Context, events event.Dispatcher) *ConnPool {
	return &ConnPool{ctx: ctx, events: events}
}

// ConnPool is a pool of connections that will reuse already connected clients
type ConnPool struct {
	clients []*Client
	events  event.Dispatcher
	ctx     context.Context
}

func (pl *ConnPool) AddClient(addr, key string) error {
	for _, cl := range pl.clients {
		if cl.addr == addr {
			return nil
		}
	}
	cl, err := NewClient(addr, key)
	if err != nil {
		return err
	}
	pl.clients = append(pl.clients, cl)
	return nil
}

// GetClient will return the client for the given addr and key
func (pl *ConnPool) GetClient(addr, key string) (*Client, error) {
	pl.events.Go("log", "attempting to find client for "+addr)
	for _, cl := range pl.clients {
		if cl.addr == addr {
			if !cl.connected {
				go cl.ConnectWithContext(pl.ctx, pl.events)
				cl.WaitForConnect()
			}
			return cl, nil
		}
	}

	pl.events.Go("log", "no client found, creating new client for "+addr)
	cl, err := NewClient(addr, key)
	if err != nil {
		return nil, err
	}
	pl.clients = append(pl.clients, cl)

	pl.events.Go("log", "attempting to conenct the client "+addr)
	go cl.ConnectWithContext(pl.ctx, pl.events)
	cl.WaitForConnect()
	pl.events.Go("log", "connected the client for "+addr)
	return cl, nil
}

func PopulatePool(pool Pool, cfg Config) error {
	switch p := pool.(type) {
	case *ConnPool:
		for _, t := range cfg.Tunnels {
			for _, k := range cfg.Keys {
				if t.Address == k.Address {
					if err := p.AddClient(k.Address, k.Private); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
