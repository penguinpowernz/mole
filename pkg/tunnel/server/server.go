package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/AlexanderGrom/go-event"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// Server represents the tunnel server
type Server struct {
	*ssh.Server
	cfg    *Config
	events event.Dispatcher
}

// NewServer will create a new tunnel server using the given config
// events dispatcher
func NewServer(cfg *Config, events event.Dispatcher) *Server {
	svr := &Server{cfg: cfg, events: events}
	svr.buildSSHServer()

	svr.SetOption(ssh.WrapConn(func(ctx ssh.Context, conn net.Conn) net.Conn {
		svr.events.Go("log", fmt.Sprintf("New connection from %s", conn.RemoteAddr().String()))
		return conn
	}))

	svr.SetOption(ssh.PublicKeyAuth(svr.IsKeyAuthorized))
	svr.SetOption(ssh.HostKeyPEM([]byte(cfg.HostKey)))
	svr.SetOption(ssh.NoPty())

	return svr
}

func (svr *Server) buildSSHServer() {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	svr.Server = &ssh.Server{
		Addr: svr.cfg.ListenPort,
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
}

// ListenAndServe will run the server until the context is done or
// the server quits for some reason
func (svr *Server) ListenAndServe(ctx context.Context) {
	go func() {
		err := svr.Server.ListenAndServe()
		svr.events.Go("log", "ERROR: "+err.Error())
		svr.Close()
	}()
	<-ctx.Done()
	svr.Close()
}

// IsKeyAuthorized is a handler for the server authentication check returning true
// if the public key is match for the given client
func (svr *Server) IsKeyAuthorized(ctx ssh.Context, key ssh.PublicKey) bool {
	svr.events.Go("log", fmt.Sprintf("incoming authentication req for %s from %s", ctx.User(), ctx.RemoteAddr().String()))
	allowed, _, _, _, _ := ssh.ParseAuthorizedKey(svr.cfg.AuthorizedKeyBytes())
	log.Printf("keychekc %s\n", string(gossh.MarshalAuthorizedKey(key)))
	log.Printf("%s\n", string(svr.cfg.AuthorizedKeyBytes()))
	return ssh.KeysEqual(key, allowed)
}
