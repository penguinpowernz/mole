package tunnel

import (
	"context"
	"io"
	"log"
	"net"
)

type Strategy func(context.Context, SSHConn) error

type SSHConn interface {
	Dial(string, string) (net.Conn, error)
	Listen(string, string) (net.Listener, error)
}

func RemoteStrategy(local, remote string) Strategy {
	return Strategy(func(ctx context.Context, conn SSHConn) error {
		l, err := conn.Listen("tcp", remote)
		if err != nil {
			return err
		}
		defer l.Close()

		go func() {
			for {
				upstream, err := l.Accept()
				if err != nil {
					break
				}

				downstream, err := net.Dial("tcp", local)
				if err != nil {
					continue
				}

				go Bridge(upstream, downstream)
			}
		}()

		<-ctx.Done()
		return nil
	})
}

func LocalStrategy(local, remote string) Strategy {
	return Strategy(func(ctx context.Context, conn SSHConn) error {
		l, err := net.Listen("tcp", local)
		if err != nil {
			return err
		}
		defer l.Close()

		go func() {
			for {
				downstream, err := l.Accept()
				if err != nil {
					break
				}

				upstream, err := conn.Dial("tcp", remote)
				if err != nil {
					continue
				}

				go Bridge(upstream, downstream)
			}
		}()

		<-ctx.Done()
		return nil
	})
}

func Bridge(upstream, downstream net.Conn) {
	upDone := make(chan struct{})
	downDone := make(chan struct{})

	// Copy localConn.Reader to sshConn.Writer
	go func() {
		_, err := io.Copy(upstream, downstream)
		if err != nil {
			log.Printf("io.Copy failed: %v", err)
		}
		close(upDone)
	}()

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		_, err := io.Copy(downstream, upstream)
		if err != nil {
			log.Printf("io.Copy failed: %v", err)
		}
		close(downDone)
	}()

	defer downstream.Close()
	defer upstream.Close()

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
