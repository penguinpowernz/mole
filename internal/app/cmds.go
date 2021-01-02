package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlecAivazis/survey"
	"github.com/gliderlabs/ssh"
	"github.com/penguinpowernz/eztunnel/internal/util"
	eztunnel "github.com/penguinpowernz/eztunnel/pkg"
)

func (app *App) doAddAuthorizedKey() {
	var newpubkey string
	survey.AskOne(&survey.Input{
		Message: "Paste the public key here:",
		Default: "",
		Help:    "(the public key you see when selecting the menu option '" + actnShowPubKey + "')",
	}, &newpubkey)

	if newpubkey == "" {
		fmt.Println("No public key specified")
		return
	}

	app.cfg.AddAuthorizedKey(newpubkey)
	app.cfg.Save()
	fmt.Println("New public key was saved to your list of authorized keys")
}

func (app *App) doAcceptConnection() {
	fmt.Println("Waiting for new connections, push CTRL+C to cancel...")
	app.svr.PublicKeyHandler = func(ctx ssh.Context, key ssh.PublicKey) bool {
		var allow bool
		survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Allow %s from %s to connect?", ctx.User(), ctx.RemoteAddr().String()),
			Default: false,
		}, &allow)

		if allow {
			app.cfg.AddAuthorizedKey(key)
			app.cfg.Save()
			fmt.Println("New public key was saved to your list of authorized keys")
		}

		return allow
	}
	defer func() { app.svr.PublicKeyHandler = app.svr.IsKeyAuthorized }()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
	)

	<-sigc
}

func (app *App) doConnect() {
	var addr, lport, rport string

	for addr == "" {
		survey.AskOne(&survey.Input{
			Message: "Enter the hostname (and port) to connect to:",
			Help:    "192.168.1.1:4443",
			Default: "",
		}, &addr)
	}

	for lport == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Enter the local port to forward:",
			Default: "",
		}, &lport); err != nil {
			return
		}
	}

	for rport == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Enter the remote port to forward to:",
			Default: "",
		}, &rport); err != nil {
			return
		}
	}

	tun := eztunnel.Tunnel{Enabled: true, Remote: rport, Local: lport, Address: addr}
	app.cfg.Tunnels = append(app.cfg.Tunnels, tun)
	app.cfg.Save()

	app.tunMgr.OpenTunnels(tun)
	util.PrintLogsUntilEnter()
}
