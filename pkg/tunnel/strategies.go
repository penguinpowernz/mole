package tunnel

import (
	"context"
	"io"
	"log"
	"net"
)

// Strategy is a tunneling strategy that can be used to do
// port forwarding over an SSH connection
type Strategy func(context.Context, SSHConn) error

type SSHConn interface {
	Dial(string, string) (net.Conn, error)
	Listen(string, string) (net.Listener, error)
}

// ReverseStrategy is a strategy for setting up a reverse port
// forward from a remote port to a local port
func ReverseStrategy(local, remote string) Strategy {
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

				go Bridge(ctx, upstream, downstream)
			}
		}()

		<-ctx.Done()
		return nil
	})
}

// LocalStrategy is a strategy for setting up a port
// forward from a local port to a remote port
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

				go Bridge(ctx, upstream, downstream)
			}
		}()

		<-ctx.Done()
		return nil
	})
}

// Bridge will mirror two active network connections using the given
// context to allow stopping the mirror
func Bridge(ctx context.Context, upstream, downstream net.Conn) {
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
		case <-ctx.Done():
			return
		case <-downDone:
			return
		case <-upDone:
			return
		}
	}
}
