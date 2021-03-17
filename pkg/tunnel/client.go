package tunnel

import (
	"context"
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
func NewClient(addr string, _privkey string) (*Client, error) {
	sshcfg := &ssh.ClientConfig{
		User:            os.Getenv("USER"),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	privkey, err := ssh.ParsePrivateKey([]byte(_privkey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %s", err)
	}

	sshcfg.Auth = append(sshcfg.Auth, ssh.PublicKeys(privkey))

	return &Client{
		addr:   addr,
		sshcfg: sshcfg,
		mu:     new(sync.Mutex),
	}, nil
}

// Client is an SSH connection to a mole server or SSH server
type Client struct {
	addr      string
	ssh       *ssh.Client
	sshcfg    *ssh.ClientConfig
	connected bool

	mu       *sync.Mutex
	deadChan chan struct{}
}

func (cl *Client) Dial(n, a string) (net.Conn, error) {
	return cl.ssh.Dial(n, a)
}

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

// Connect will connect to the server returning an error
// if the connect failed
func (cl *Client) Connect() (err error) {
	cl.ssh, err = ssh.Dial("tcp", cl.addr, cl.sshcfg)
	if err != nil {
		return err
	}

	return nil
}
