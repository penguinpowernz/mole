package tunnel

import (
	"context"
	"errors"
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

	mu       *sync.Mutex `json:"-"`
	strategy Strategy
}

// NewTunnelFromOpts will create a new tunnel from the given options
func NewTunnelFromOpts(opts ...Option) (*Tunnel, error) {
	t := &Tunnel{}
	for _, opt := range opts {
		if err := opt(t); err != nil {
			return t, err
		}
	}

	if t.strategy == nil {
		t.strategy = LocalStrategy(t.Local, t.Remote)
		if t.Reverse {
			t.strategy = ReverseStrategy(t.Local, t.Remote)
		}
	}

	return t, nil
}

// Open will "open" the tunnel, by listening for new connections coming into
// the local port, and then hooking them up to the remote port on the fly
func (tun *Tunnel) Open(ctx context.Context, cl SSHConn) (err error) {
	if tun.strategy == nil {
		return errors.New("no strategy added to tunnel")
	}

	if tun.mu == nil {
		tun.mu = new(sync.Mutex)
	}

	tun.mu.Lock()
	defer tun.mu.Unlock()

	if tun.IsOpen {
		return nil
	}

	tun.normalizePorts()

	doneChan := make(chan bool)
	go func() {
		_ = tun.strategy(ctx, cl) // TODO: print the error
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
