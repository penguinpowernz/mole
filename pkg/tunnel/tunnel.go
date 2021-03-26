package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/AlexanderGrom/go-event"
)

// Dialer is a function that will dial a remote SSH server
type Dialer func(string, string) (net.Conn, error)

// Tunnel represents the tunnel as it appears in the config, but
// as an object that will do the actual tunnel connection
type Tunnel struct {
	addr string

	Local      string `json:"local_port"`
	Remote     string `json:"remote_port"`
	Disabled   bool   `json:"disabled"`
	Reverse    bool   `json:"reverse"`
	ReverseDef string `json:"R"`
	LocalDef   string `json:"L"`

	IsOpen bool `json:"-"`

	mu       *sync.Mutex
	strategy Strategy
	doneChan chan bool
}

type Tunnels []*Tunnel

func (tuns Tunnels) Open() []*Tunnel {
	tunss := []*Tunnel{}
	for _, t := range tuns {
		if t.IsOpen {
			tunss = append(tunss, t)
		}
	}
	return tunss
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

// UnmarshalJSON will unmarshal the individual tunnel config and
// setup the relevant config derived fields so that it is ready to use
func (tun *Tunnel) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, tun); err != nil {
		return err
	}

	if tun.LocalDef != "" {
		if err := PFD(tun.LocalDef)(tun); err != nil {
			return err
		}
		return nil
	}

	if tun.ReverseDef != "" {
		if err := PFD(tun.ReverseDef)(tun); err != nil {
			return err
		}
		tun.Reverse = true
		return nil
	}

	if tun.strategy == nil {
		tun.strategy = LocalStrategy(tun.Local, tun.Remote)
		if tun.Reverse {
			tun.strategy = ReverseStrategy(tun.Local, tun.Remote)
		}
	}

	return nil
}

// KeepOpen will open the tunnel and keep it open if it closes
func (tun *Tunnel) KeepOpen(ctx context.Context, cl SSHConn, ev event.Dispatcher) {
	for {
		if err := tun.Open(ctx, cl); err != nil {
			ev.Go("log", fmt.Sprintf("ERROR: failed to open tunnel for %s: %s", tun.Name(), err))
			time.Sleep(time.Second)
			continue
		}

		ev.Go("log", fmt.Sprintf("tunnel opened: %s", tun.Name()))

		select {
		case <-tun.doneChan:
			ev.Go("log", fmt.Sprintf("tunnel closed: %s", tun.Name()))
			time.Sleep(time.Second)
			continue
		case <-ctx.Done():
			ev.Go("log", fmt.Sprintf("tunnel done: %s", tun.Name()))
			return
		}
	}
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

	tun.doneChan = make(chan bool)
	go func() {
		if err := tun.strategy(ctx, cl); err != nil && ctx.Err() == nil {
			log.Printf("ERROR: %s stopped: %s", tun, err) // only print the error if the ctx wasn't quit
		}
		close(tun.doneChan)
	}()

	tun.IsOpen = true

	go func() {
		<-tun.doneChan
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
	return strings.ReplaceAll(tun.String(), " ", "")
}

func (tun *Tunnel) String() string {
	dir := "-->"
	if tun.Reverse {
		dir = "<--"
	}
	return fmt.Sprintf("%50s [    %30s  %s  %-30s     ]", tun.addr, tun.Local, dir, tun.Remote)
}
