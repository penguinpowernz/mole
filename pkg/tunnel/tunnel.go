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
	Reverse bool   `json:"reverse"`
	Disabled   bool   `json:"disabled"`

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

func normalizePort(p string) string {
	h := "localhost"
	if p[0] == ':' || (p[0] != ':' && !strings.Contains(p, ":")) {
		if p[0] != ':' {
			h += ":"
		}
		p = h + p
	}
	return p
}

func (tun *Tunnel) normalizePorts() {
	tun.Remote = normalizePort(tun.Remote)
	tun.Local = normalizePort(tun.Local)
}

// Name will return the name of this tunnel
func (tun *Tunnel) Name() string {
	return tun.Local + ":" + tun.Remote
}
