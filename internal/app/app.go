package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlecAivazis/survey"
	"github.com/gliderlabs/ssh"
	"github.com/penguinpowernz/mole/pkg/tunnel/server"
)

// InteractivelyAcceptPublicKeys will change the server auth function so that it
// explicitly requests acceptances from the console for each incoming request and
// saves those public keys to the config file
func InteractivelyAcceptPublicKeys(svr *server.Server, cfg *server.Config) {
	fmt.Println("Waiting for new connections, push CTRL+C to cancel...")
	svr.PublicKeyHandler = func(ctx ssh.Context, key ssh.PublicKey) bool {
		// don't ask about known ones
		if svr.IsKeyAuthorized(ctx, key) {
			return true
		}

		var allow bool

		survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Allow %s from %s to connect?", ctx.User(), ctx.RemoteAddr().String()),
			Default: false,
		}, &allow)

		if allow {
			cfg.AddAuthorizedKey(key)
			cfg.Save()
			fmt.Println("New public key was saved to your list of authorized keys")
		}

		return allow
	}
	defer func() { svr.PublicKeyHandler = svr.IsKeyAuthorized }()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
	)

	<-sigc
}
