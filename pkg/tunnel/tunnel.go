package tunnel

import (
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

	lstnr  net.Listener `json:"-"`
	IsOpen bool         `json:"-"`

	dialer Dialer           `json:"-"`
	mu     *sync.Mutex      `json:"-"`
	ev     event.Dispatcher `json:"-"`
}

// NewTunnelFromPool will create a new tunnel from the given address, remote/local ports
// and private key, using the given pool to obtain a client connection
func NewTunnelFromPool(pool Pool, addr, remote, local, key string) (*Tunnel, error) {
	t := &Tunnel{
		Address: addr,
		Local:   local,
		Remote:  remote,
		Enabled: true,
	}

	err := t.furnish(pool, key)
	return nil, err
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
		tun := &t
		tun.furnish(pool, cfg.PrivateKey)
		tuns = append(tuns, tun)
	}

	return tuns, nil
}

func (tun *Tunnel) furnish(pool Pool, key string) error {
	cl, err := pool.GetClient(tun.Address, key)
	if err != nil {
		return err
	}
	tun.mu = new(sync.Mutex)
	tun.dialer = cl.dialerFunc()
	return nil
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

	tun.normalizePorts()

	tun.lstnr, err = net.Listen("tcp", tun.Local)
	if err != nil {
		log.Println("Failed to open port for local listener: ", err)
		return
	}
	tun.IsOpen = true

	go tun.listenForConnections()
	return nil
}

// wait for someone to connect to the port and then pass that off to be hooked up to the remote port
func (tun *Tunnel) listenForConnections() {
	log.Println("listening for connections on ", tun.Local)
	for {
		log.Println("waiting for new connection")
		local, err := tun.lstnr.Accept() // TODO: will this close when the listener is closed?
		if err != nil {
			log.Println("Failed to accept listeners conn: ", err)
		}

		go tun.handleConnection(local)
	}
}

func (tun *Tunnel) normalizePorts() {
	if tun.Remote[0] == ':' || (tun.Remote[0] != ':' && !strings.Contains(tun.Remote, ":")) {
		tun.Remote = "localhost:" + tun.Remote
	}

	if tun.Local[0] == ':' || (tun.Local[0] != ':' && !strings.Contains(tun.Local, ":")) {
		tun.Local = "localhost:" + tun.Local
	}
}

// someone requested data from the local port, so use the connection to them and hook it
// to the remote ports connection
func (tun *Tunnel) handleConnection(local net.Conn) {
	tun.ev.Go("log", "new connection requested to remote port "+tun.Remote)
	remote, err := tun.dialer("tcp", tun.Remote)
	if err != nil {
		log.Println("Failed to open port to remote: ", err)
		local.Close()
		return
	}
	tun.ev.Go("log", "new connection opened to remote port "+tun.Remote)

	upDone := make(chan struct{})
	downDone := make(chan struct{})

	// Copy localConn.Reader to sshConn.Writer
	go func() {
		_, err := io.Copy(remote, local)
		if err != nil {
			log.Printf("io.Copy failed: %v", err)
		}
		close(upDone)
	}()

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		_, err := io.Copy(local, remote)
		if err != nil {
			log.Printf("io.Copy failed: %v", err)
		}
		close(downDone)
	}()

	tun.ev.Go("log", "tunnel port "+tun.Name()+" was opened, copying data across")

	defer local.Close()
	defer remote.Close()
	defer tun.ev.Go("log", "data transfer complete")

	for {
		select {
		// TODO: add a context done check here somehow?  Otherwise the connection may persist or hold up the closing of the client
		case <-downDone:
			return
		case <-upDone:
			return
		}
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
