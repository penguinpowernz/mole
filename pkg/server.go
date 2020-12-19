package eztunnel

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	*ssh.Server
	cfg *Config
}

func (svr *Server) ListenAndServe(ctx context.Context) {
	go func() {
		fmt.Println("ERROR:", svr.Server.ListenAndServe())
		svr.Close()
	}()
	<-ctx.Done()
	svr.Close()
}

func (svr *Server) IsKeyAuthorized(ctx ssh.Context, key ssh.PublicKey) bool {
	fmt.Printf("incoming authentication req for %s from %s", ctx.User(), ctx.RemoteAddr().String())
	allowed, _, _, _, _ := ssh.ParseAuthorizedKey(svr.cfg.AuthorizedKeyBytes())
	return ssh.KeysEqual(key, allowed)
}

func NewServer(cfg *Config) *Server {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	svr := &Server{cfg: cfg}
	server := ssh.Server{
		Addr: cfg.ListenPort,
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			log.Println("Accepted forward", dhost, dport, "from", ctx.RemoteAddr().String())
			return true
		}),
		Handler: ssh.Handler(func(s ssh.Session) {
			log.Printf("%+v\n", s.RemoteAddr().String())
			select {}
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			log.Println("attempt to bind", host, port, "granted")
			return true
		}),
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
				log.Println("directtcp", srv.Addr, conn.LocalAddr(), conn.RemoteAddr())
				ssh.DirectTCPIPHandler(srv, conn, newChan, ctx)
				log.Println("directtcp", srv.Addr, conn.LocalAddr(), conn.RemoteAddr())
			},
			"session": ssh.DefaultSessionHandler,
			"iotunnel": func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
				log.Println("iotunnel", srv.Addr, conn.LocalAddr(), conn.RemoteAddr())
				log.Println("arbdata", string(newChan.ExtraData()))
				outch, inch, _ := newChan.Accept()
				r := <-inch
				log.Println(r.Type, string(r.Payload))
				outch.Write([]byte(`see ya later aligator`))
				outch.Close()
			},
		},
	}

	svr.Server = &server
	svr.SetOption(ssh.PublicKeyAuth(svr.IsKeyAuthorized))

	server.SetOption(ssh.WrapConn(func(ctx ssh.Context, conn net.Conn) net.Conn {
		log.Printf("New connection from %s", conn.RemoteAddr().String())
		return conn
	}))

	server.SetOption(ssh.HostKeyPEM([]byte(cfg.HostKey)))

	server.SetOption(ssh.NoPty())

	return svr
}
