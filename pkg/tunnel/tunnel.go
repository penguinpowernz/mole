package tunnel

import (
	"context"
	"net"
	"strings"
	"sync"
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

	IsOpen bool `json:"-"`

	mu *sync.Mutex `json:"-"`
}

// Open will "open" the tunnel, by listening for new connections coming into
// the local port, and then hooking them up to the remote port on the fly
func (tun *Tunnel) Open(ctx context.Context, cl SSHConn) (err error) {
	if tun.mu == nil {
		tun.mu = new(sync.Mutex)
	}

	tun.mu.Lock()
	defer tun.mu.Unlock()

	if tun.IsOpen {
		return nil
	}

	tun.normalizePorts()

	strategy := LocalStrategy(cl, tun.Local, tun.Remote)
	if tun.Reverse {
		strategy = RemoteStrategy(cl, tun.Local, tun.Remote)
	}

	doneChan := make(chan bool)
	go func() {
		_ = strategy(ctx) // TODO: print the error
		close(doneChan)
	}()

	tun.IsOpen = true

	go func() {
		select {
		case <-ctx.Done():
		case <-doneChan:
		}
		tun.IsOpen = false
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
