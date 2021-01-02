package tunnel

import (
	"io"
	"log"
	"net"
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

	lstnr  net.Listener `json:"-"`
	local  net.Conn     `json:"-"`
	remote net.Conn     `json:"-"`
	IsOpen bool         `json:"-"`

	dialer Dialer
	mu     *sync.Mutex
	ev     event.Dispatcher
}

// NewTunnelFromPool will create a new tunnel from the given address, remote/local ports
// and private key, using the given pool to obtain a client connection
func NewTunnelFromPool(pool Pool, addr, remote, local, key string) (*Tunnel, error) {
	cl, err := pool.GetClient(addr, key)
	if err != nil {
		return nil, err
	}

	return &Tunnel{
		Address: addr,
		Local:   local,
		Remote:  remote,
		Enabled: true,
		mu:      new(sync.Mutex),
		dialer:  cl.DialerFunc(),
	}, nil
}

// NewTunnel will create a new tunnel from the given address, remote/local ports
// and private key, using the default global pool to obtain a client connection
func NewTunnel(addr, remote, local, key string) (*Tunnel, error) {
	return NewTunnelFromPool(DefaultPool, addr, remote, local, key)
}

// NewTunnelsFromConfig will create a bunch of tunnels from what is specified in
// the given config using the default global pool to obtain a client connection
func NewTunnelsFromConfig(cfg Config) ([]*Tunnel, error) {
	return NewTunnelsFromConfigAndPool(DefaultPool, cfg)
}

// NewTunnelsFromConfigAndPool will create a bunch of tunnels from what is specified in
// the given config using the given pool to obtain a client connection
func NewTunnelsFromConfigAndPool(pool Pool, cfg Config) ([]*Tunnel, error) {
	tuns := []*Tunnel{}

	for _, t := range cfg.Tunnels {
		cl, err := pool.GetClient(t.Address, cfg.PrivateKey)
		if err != nil {
			return tuns, err
		}

		tun := &t
		tun.mu = new(sync.Mutex)
		tun.dialer = cl.DialerFunc()
		tuns = append(tuns, tun)
	}

	return tuns, nil
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
func (tun *Tunnel) Open() (err error) {
	if tun.mu == nil {
		tun.mu = new(sync.Mutex)
	}

	tun.mu.Lock()
	defer tun.mu.Unlock()

	if tun.IsOpen {
		return nil
	}

	tun.lstnr, err = net.Listen("tcp", tun.Local)
	if err != nil {
		log.Println("Failed to open port for local listener: ", err)
		return
	}
	log.Println("listening for connections on ", tun.Local)

	go func() {
		for {
			log.Println("waiting for new connection")
			tun.local, err = tun.lstnr.Accept()
			if err != nil {
				log.Println("Failed to accept listeners conn: ", err)
			}

			tun.ev.Go("log", "new connection requested to remote port "+tun.Remote)
			tun.remote, err = tun.dialer("tcp", tun.Remote)
			if err != nil {
				log.Println("Failed to open port to remote: ", err)
			}
			tun.ev.Go("log", "new connection opened to remote port "+tun.Remote)

			upDone := make(chan struct{})
			downDone := make(chan struct{})

			// Copy localConn.Reader to sshConn.Writer
			go func() {
				_, err := io.Copy(tun.remote, tun.local)
				if err != nil {
					log.Printf("io.Copy failed: %v", err)
				}
				close(upDone)
			}()

			// Copy sshConn.Reader to localConn.Writer
			go func() {
				_, err := io.Copy(tun.local, tun.remote)
				if err != nil {
					log.Printf("io.Copy failed: %v", err)
				}
				close(downDone)
			}()

			tun.IsOpen = true
			tun.ev.Go("log", "tunnel port "+tun.Name()+" was opened, copying data across")
			<-upDone
			tun.ev.Go("log", "data was transferred")
			tun.remote.Close()
			tun.local.Close()
			<-downDone
			// tun.IsOpen = false
			// tun.ev.Go("log", "tunnel port "+tun.Name()+" was closed")
			// tun.Close()

		}
	}()

	tun.IsOpen = true
	return nil
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

	if tun.local != nil {
		go tun.local.Close()
	}

	if tun.remote != nil {
		go tun.remote.Close()
	}

	tun.IsOpen = false
}
