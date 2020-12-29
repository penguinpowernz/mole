package tunnel

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/AlexanderGrom/go-event"
	"golang.org/x/crypto/ssh"
)

// type ClientManager struct {
// 	clients []*Client
// 	events  event.Dispatcher
// 	cfg     *Config
// }

// func (mgr ClientManager) Listen() {
// 	mgr.events.On("connect.client", func(addr string) error {

// 		cl, ok := mgr.FindClientByAddr(addr)
// 		if ok {
// 			mgr.events.Go("client.connected", cl)
// 			return nil
// 		}

// 		cl, err := NewClient(addr, mgr.cfg.PrivateKey)
// 		if err != nil {
// 			return err
// 		}

// 		if err = cl.Connect(); err != nil {
// 			return err
// 		}

// 		mgr.events.Go("client.connected", cl)

// 		mgr.clients = append(mgr.clients, cl)
// 		return nil
// 	})

// }

// func (mgr *ClientManager) FindClientByAddr(addr string) (*Client, bool) {
// 	for _, cl := range mgr.clients {
// 		if cl.addr == addr {
// 			return cl, true
// 		}
// 	}
// 	return nil, false
// }

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
	}, nil
}

type Client struct {
	addr      string
	ssh       *ssh.Client
	sshcfg    *ssh.ClientConfig
	connected bool

	mu       *sync.Mutex
	deadChan chan struct{}
}

func (cl *Client) DialerFunc() Dialer {
	return cl.ssh.Dial
}

func (cl *Client) WaitForConnect() {
	for {
		if cl.connected {
			return
		}
		time.Sleep(time.Second / 5)
	}
}

func (cl *Client) IsForMe(tun Tunnel) bool {
	return tun.Address == cl.addr
}
func (cl *Client) Close() (err error) {
	return cl.ssh.Close()
}

func (cl *Client) ConnectWithContext(ctx context.Context, events event.Dispatcher) {
	t := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-t.C:
			if !cl.connected {
				if err := cl.Connect(); err != nil {
					events.Go("error", fmt.Errorf("failed to connect to %s: %s", cl.addr, err))
					continue
				}

				cl.connected = true

				go func() {
					if err := cl.ssh.Wait(); err != nil {
						events.Go("error", fmt.Errorf("client %s disconnected: %s", cl.addr, err))
					}
					cl.deadChan <- struct{}{}
				}()

				events.Go("log", "client "+cl.addr+" was connected")
				events.Go("client.connected", cl)
			}
		case <-ctx.Done():
			events.Go("log", fmt.Sprintf("context done for client %s", cl.addr))
			cl.Close()
			return

		case <-cl.deadChan:
			cl.connected = false
			events.Go("client.disconnected", cl)
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
