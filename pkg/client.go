package eztunnel

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/ssh"
)

func NewClient(addr string, _privkey string) (*Client, error) {
	sshcfg := &ssh.ClientConfig{
		User: "username",
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	privkey, err := ssh.ParsePrivateKey([]byte(_privkey))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key: %s", err)
	}

	sshcfg.Auth = append(sshcfg.Auth, ssh.PublicKeys(privkey))

	return &Client{
		sshcfg: sshcfg,
		mu:     new(sync.Mutex),
	}, nil
}

type Client struct {
	addr   string
	ssh    *ssh.Client
	sshcfg *ssh.ClientConfig
	tuns   map[string]*Tunnel

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

func (cl *Client) Connect() (err error) {
	cl.ssh, err = ssh.Dial("tcp", cl.addr, cl.sshcfg)
	if err != nil {
		return err
	}

	return nil
}
