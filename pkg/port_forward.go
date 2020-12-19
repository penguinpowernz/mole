package eztunnel

import (
	"errors"
	"net"
)

var ErrNotEnabled = errors.New("tunnel not enabled")

type Dialer func(string, string) (net.Conn, error)

// func FowardPort(dialer Dialer, local, remote string) (PortForward, error) {
// 	pf := NewPortForward(dialer, local, remote)
// 	err := pf.Open()
// 	return pf, err
// }

// func NewPortForward(dialer Dialer, local, remote string) *PortForward {
// 	return &PortForward{dialer: dialer, Lport: local, Rport: remote}
// }

// type PortForward struct {
// 	dialer  Dialer
// 	lstnr   net.Listener
// 	local   net.Conn
// 	remote  net.Conn
// 	Lport   string `json:"local"`
// 	Rport   string `json:"remote"`
// 	isOpen  bool
// 	Enabled bool   `json:"enabled"`
// 	Address string `json:"address"`
// }

// func (pf *PortForward) Open() (err error) {
// 	pf.lstnr, err = net.Listen("tcp", pf.Lport)
// 	if err != nil {
// 		log.Println("Failed to open port for local listener: ", err)
// 		return
// 	}

// 	go func() {
// 		for {
// 			pf.local, err = pf.lstnr.Accept()
// 			if err != nil {
// 				log.Println("Failed to accept listeners conn: ", err)
// 			}

// 			pf.remote, err = pf.dialer("tcp", pf.Rport)
// 			if err != nil {
// 				log.Println("Failed to open port to remote: ", err)
// 			}

// 			forward(pf.remote, pf.local)
// 		}
// 	}()

// 	pf.isOpen = true

// 	return
// }

// func (pf *PortForward) Close() {
// 	pf.lstnr.Close()
// 	go pf.local.Close()
// 	go pf.remote.Close()
// 	pf.isOpen = false
// }
