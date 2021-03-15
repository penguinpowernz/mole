package tunnel

import (
	"context"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/AlexanderGrom/go-event"
)

// Dialer is a function that will dial a remote SSH server
type Dialer func(string, string) (net.Conn, error)

// Tunnel represents the tunnel as it appears in the config, but
// as an object that will do the actual tunnel connection
type Tunnel struct {
	Address string `json:"address"`
	Local   string `json:"local_port"`
	Remote  string `json:"remote_port"`
	Enabled bool   `json:"enabled"`
	Reverse bool   `json:"reverse"`

	lstnr  net.Listener `json:"-"`
	IsOpen bool         `json:"-"`

	dialer Dialer           `json:"-"`
	mu     *sync.Mutex      `json:"-"`
	ev     event.Dispatcher `json:"-"`
}

// Listen will listen to event from the dispatcher
func (tun *Tunnel) Listen(events event.Dispatcher) {
	tun.ev = events
	// events.On("client.connected", func(cl *Client) error {
	// 	if cl.addr == tun.Address {
	// 		if tun.Enabled {
	// 			if err := tun.Open(); err != nil {
	// 				events.Go("error", err)
	// 			}
	// 		}
	// 	}
	// 	return nil
	// })

	events.On("tunnel.disable", func(t Tunnel) error {
		if tun.Name() == t.Name() {
			tun.Close()
			tun.Enabled = false
		}
		return nil
	})

	events.On("tunnel.enable", func(t Tunnel) error {
		if tun.Name() == t.Name() {
			tun.Enabled = true
			event.Go("connect.client", tun.Address)
		}
		return nil
	})

	// event.Go("connect.client", tun.Address)
}

// Open will "open" the tunnel, by listening for new connections coming into
// the local port, and then hooking them up to the remote port on the fly
func (tun *Tunnel) Open(ctx context.Context) (err error) {
	if tun.mu == nil {
		tun.mu = new(sync.Mutex)
	}

	tun.mu.Lock()
	defer tun.mu.Unlock()

	if tun.IsOpen {
		return nil
	}

	tun.normalizePorts()

	}
	tun.IsOpen = true


	go func() {
		<-ctx.Done()
		tun.Close()
	}()

	return nil
}

func (tun *Tunnel) normalizePorts() {
	if tun.Remote[0] == ':' || (tun.Remote[0] != ':' && !strings.Contains(tun.Remote, ":")) {
		tun.Remote = "localhost:" + tun.Remote
	}

	if tun.Local[0] == ':' || (tun.Local[0] != ':' && !strings.Contains(tun.Local, ":")) {
		tun.Local = "localhost:" + tun.Local
	}
}

// Name will return the name of this tunnel
func (tun *Tunnel) Name() string {
	return tun.Local + ":" + tun.Remote
}

// Close will close this tunnel by closing the listener
func (tun *Tunnel) Close() {
	if tun.lstnr != nil {
		tun.lstnr.Close()
	}
	tun.IsOpen = false
}
