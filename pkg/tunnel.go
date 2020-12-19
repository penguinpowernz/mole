package eztunnel

import (
	"io"
	"log"
	"net"
)

type Tunnel struct {
	Address string `json:"address"`
	Local   string `json:"local_port"`
	Remote  string `json:"remote_port"`
	Enabled bool   `json:"enabled"`

	lstnr  net.Listener `json:"-"`
	local  net.Conn     `json:"-"`
	remote net.Conn     `json:"-"`
	IsOpen bool         `json:"-"`
}

func (tun *Tunnel) Open(dialer Dialer) (err error) {
	tun.lstnr, err = net.Listen("tcp", tun.Local)
	if err != nil {
		log.Println("Failed to open port for local listener: ", err)
		return
	}

	go func() {
		for {
			tun.local, err = tun.lstnr.Accept()
			if err != nil {
				log.Println("Failed to accept listeners conn: ", err)
			}

			tun.remote, err = dialer("tcp", tun.Remote)
			if err != nil {
				log.Println("Failed to open port to remote: ", err)
			}

			upDone := make(chan struct{})
			downDone := make(chan struct{})

			// Copy localConn.Reader to sshConn.Writer
			go func() {
				_, err := io.Copy(tun.remote, tun.local)
				if err != nil {
					log.Printf("io.Copy failed: %v", err)
				}
				log.Println("done with copy")
				close(upDone)
			}()

			// Copy sshConn.Reader to localConn.Writer
			go func() {
				_, err := io.Copy(tun.local, tun.remote)
				if err != nil {
					log.Printf("io.Copy failed: %v", err)
				}
				log.Println("done with copy2")
				close(downDone)
			}()

			<-upDone
			<-downDone
			tun.IsOpen = false
			tun.Close()
		}
	}()

	tun.IsOpen = true
	return nil
}

func (tun *Tunnel) Name() string {
	return tun.Local + ":" + tun.Remote
}

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
