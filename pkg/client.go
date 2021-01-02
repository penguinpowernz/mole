package eztunnel

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/AlexanderGrom/go-event"
	"golang.org/x/crypto/ssh"
)

type ClientManager struct {
	clients []*Client
	events  event.Dispatcher
	cfg     *Config
}

func (mgr ClientManager) Listen() {
	mgr.events.On("connect.client", func(addr string) error {

		cl, ok := mgr.FindClientByAddr(addr)
		if ok {
			mgr.events.Go("client.connected", cl)
			return nil
		}

		cl, err := NewClient(addr, mgr.cfg.PrivateKey)
		if err != nil {
			return err
		}

		if err = cl.Connect(); err != nil {
			return err
		}

		mgr.events.Go("client.connected", cl)

		mgr.clients = append(mgr.clients, cl)
		return nil
	})

}

func (mgr *ClientManager) FindClientByAddr(addr string) (*Client, bool) {
	for _, cl := range mgr.clients {
		if cl.addr == addr {
			return cl, true
		}
	}
	return nil, false
}

func NewClient(addr string, _privkey string) (*Client, error) {
	sshcfg := &ssh.ClientConfig{
		User:            os.Getenv("USER") + ":" + func() string { s, _ := os.Hostname(); return s }(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	privkey, err := ssh.ParsePrivateKey([]byte(_privkey))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key: %s", err)
	}

	sshcfg.Auth = append(sshcfg.Auth, ssh.PublicKeys(privkey))

	return &Client{
		addr:   addr,
		sshcfg: sshcfg,
		mu:     new(sync.Mutex),
		tuns:   map[string]*Tunnel{},
	}, nil
}

type Client struct {
	addr      string
	ssh       *ssh.Client
	sshcfg    *ssh.ClientConfig
	tuns      map[string]*Tunnel
	connected bool

	mu *sync.Mutex
}

func (cl *Client) FindOrAddTunnel(_tun *Tunnel) (*Tunnel, bool) {
	if !cl.IsForMe(*_tun) {
		return nil, false
	}

	cl.mu.Lock()
	defer cl.mu.Unlock()

	_, found := cl.tuns[_tun.Name()]
	if !found {
		cl.tuns[_tun.Name()] = _tun
	}
	return cl.tuns[_tun.Name()], true
}

func (cl *Client) OpenTunnel(_tun Tunnel) (err error) {
	tun, ok := cl.FindOrAddTunnel(&_tun)
	if !ok {
		return nil
	}

	if !tun.Enabled {
		return nil
	}

	if tun.IsOpen {
		return nil
	}

	if err := tun.Open(cl.ssh.Dial); err != nil {
		return err
	}

	return nil
}

func (cl *Client) IsForMe(tun Tunnel) bool {
	return tun.Address == cl.addr
}

func (cl *Client) CloseTunnel(_tun Tunnel) {
	tun, ok := cl.FindOrAddTunnel(&_tun)
	if !ok {
		return
	}

	tun.Close()
}

func (cl *Client) Close() (err error) {
	for _, tun := range cl.tuns {
		tun.Close()
	}

	return cl.ssh.Close()
}

func (cl *Client) ConnectWithContext(ctx context.Context, events event.Dispatcher) {
	deadChan := make(chan bool, 1)
	t := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-t.C:
			if !cl.connected {
				if err := cl.Connect(); err != nil {
					events.Go("error", fmt.Errorf("failed to connect to %s: %s", cl.addr, err))
					continue
				}

				go func() {
					if err := cl.ssh.Wait(); err != nil {
						events.Go("error", fmt.Errorf("client %s disconnected: %s", cl.addr, err))
					}
					deadChan <- true
				}()

				events.Go("client.connected", cl)
			}
		case <-ctx.Done():
			events.Go("log", fmt.Errorf("context done for client %s", cl.addr))
			cl.Close()
			return

		case <-deadChan:

		}
	}
}

func (cl *Client) Connect() (err error) {
	cl.ssh, err = ssh.Dial("tcp", cl.addr, cl.sshcfg)
	if err != nil {
		return err
	}

	return nil
}
