package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/AlexanderGrom/go-event"
	"golang.org/x/crypto/ssh"
)

// NewClient will create a new client that will connect to the given address
// and authenticate using the given private key text.  It will return an error
// if the private key could not be parsed
func NewClient(addr string, _privkey string, hostkeys ...string) (*Client, error) {
	cl := &Client{Address: addr}
	return cl, cl.init()
}

// Client is an SSH connection to a mole server or SSH server
type Client struct {
	ssh       *ssh.Client
	sshcfg    *ssh.ClientConfig
	connected bool

	Address string    `json:"address"`
	Private string    `json:"private"`
	Public  string    `json:"public"`
	Host    string    `json:"host"`
	Tunnels []*Tunnel `json:"tunnels"`

	mu       *sync.Mutex
	deadChan chan struct{}
}

func (cl *Client) init() error {
	sshcfg := &ssh.ClientConfig{
		User:            os.Getenv("USER"),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if cl.Host != "" {
		k, err := ssh.ParsePublicKey([]byte(cl.Host))
		if err != nil {
			return fmt.Errorf("couldn't update hostkey for %s: %s", cl.Address, err)
		}
		sshcfg.HostKeyCallback = ssh.FixedHostKey(k)
	}

	privkey, err := ssh.ParsePrivateKey([]byte(cl.Private))
	if err != nil {
		return fmt.Errorf("failed to parse key for %s: %s", cl.Address, err)
	}

	sshcfg.Auth = append(sshcfg.Auth, ssh.PublicKeys(privkey))
	cl.sshcfg = sshcfg
	cl.mu = new(sync.Mutex)
	return nil
}

// UnmarshalJSON will unmmarshal the individual client configuration
// and initialize the fields on it so that it is ready to use
func (cl *Client) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, cl); err != nil {
		return err
	}
	return cl.init()
}

// HasTunnels will return true if the client has any tunnels that are enabled
func (cl *Client) HasTunnels() bool {
	var yes bool
	for _, t := range cl.Tunnels {
		if !t.Disabled {
			yes = true
		}
	}
	return yes
}

// Dial will dial a port on the remote server
func (cl *Client) Dial(n, a string) (net.Conn, error) {
	return cl.ssh.Dial(n, a)
}

// Listen will open a listener to a port on the remote server
func (cl *Client) Listen(n, a string) (net.Listener, error) {
	return cl.ssh.Listen(n, a)
}

// WaitForConnect will block until the client is connected
func (cl *Client) WaitForConnect() {
	for {
		if cl.connected {
			return
		}
		time.Sleep(time.Second / 5)
	}
}

// Close will close the client connections
func (cl *Client) Close() (err error) {
	return cl.ssh.Close()
}

// ConnectWithContext will connect using the given context to signal when to disconnect or stop
// trying to connect.  This will loop to continuously attempt to connect to the tunnel
func (cl *Client) ConnectWithContext(ctx context.Context, events event.Dispatcher) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if cl.connected {
		return
	}

	t := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-t.C:
			if !cl.connected {
				if err := cl.Connect(); err != nil {
					events.Go("error", fmt.Errorf("failed to connect to %s: %s", cl.Address, err))
					continue
				}

				cl.connected = true

				go func() {
					if err := cl.ssh.Wait(); err != nil {
						events.Go("error", fmt.Errorf("client %s disconnected: %s", cl.Address, err))
					}
					cl.deadChan <- struct{}{}
				}()

				events.Go("log", "client "+cl.Address+" was connected")
				events.Go("client.connected", cl)
			}
		case <-ctx.Done():
			events.Go("log", fmt.Sprintf("context done for client %s", cl.Address))
			cl.Close()
			return

		case <-cl.deadChan:
			cl.connected = false
			events.Go("client.disconnected", cl)
		}
	}
}

// Connect will connect to the server returning an error
// if the connect failed
func (cl *Client) Connect() (err error) {
	cl.ssh, err = ssh.Dial("tcp", cl.Address, cl.sshcfg)
	if err != nil {
		return err
	}

	return nil
}

// OpenTunnels will connect the client and open any enabled tunnels the client
// has.  If all the client has no tunnels or they are all disabled, this method
// is a no op
func (cl *Client) OpenTunnels(ctx context.Context, ev event.Dispatcher) {
	if !cl.HasTunnels() {
		return
	}

	go cl.ConnectWithContext(ctx, ev)
	ev.Go("log", fmt.Sprintf("waiting for %s to connect", cl.Address))
	cl.WaitForConnect()

	for _, tun := range cl.Tunnels {
		if tun.Disabled {
			continue
		}

		tun.addr = cl.Address // addr only used for logging purpose
		go tun.KeepOpen(ctx, cl, ev)
	}
	ev.Go("log", fmt.Sprintf("forked off all tunnel managers for %s", cl.Address))
}
