package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey"
	"github.com/AlecAivazis/survey/terminal"
	"github.com/gliderlabs/ssh"
	eztunnel "github.com/penguinpowernz/eztunnel/pkg"
)

var (
	actnStartServer = "Start listening for incoming connections"
	actnStopServer  = "Stop listening for incoming connections"
	actnShowPubKey  = "Show public key"
	actnConnect     = "Create a tunnel connection to a remote server"
	// actnGenerateHostKey  = "Generate host key"
	// actnGeneratePubKey   = "Generate public key"
	actnAddAuthorizedKey = "Add an authorized public key from another eztunnel instance"
	actnAcceptConnection = "Interactively accept connection requests (useful for setting up)"
	actnWatchLogs        = "Watch logs"
	actnDumpConfig       = "Dump config"
	actnDumpConfigFile   = "Dump config file"
	actnQuit             = "Quit"
)

type appMgr struct {
	cfg *eztunnel.Config

	svr          *eztunnel.Server
	svrStop      func()
	svrIsRunning bool

	tunMgr *eztunnel.TunnelManager
}

func (app *appMgr) showMenu() {
	for {
		var actn string
		serverToggle := actnStartServer
		if app.svrIsRunning {
			serverToggle = actnStopServer
		}
		log.Println("running", app.svrIsRunning)

		menu := []string{
			actnWatchLogs,
			actnConnect,
			actnAcceptConnection,
			serverToggle,
			actnAddAuthorizedKey,
			actnShowPubKey,
			actnDumpConfig,
			actnDumpConfigFile,
			actnQuit,
			// actnGenerateHostKey,
			// actnGeneratePubKey,
		}

		err := survey.AskOne(&survey.Select{
			Message: "What to do?",
			Options: menu,
		}, &actn)

		if err == terminal.InterruptErr {
			return
		} else if err != nil {
			panic(err)
		}

		switch actn {

		case actnWatchLogs:
			fmt.Println("Push enter to show the menu, otherwise logs will be printed")
			bufio.NewReader(os.Stdin).ReadBytes('\n')

		case actnConnect:
			app.doConnect()

		case actnAcceptConnection:
			if !app.svrIsRunning {
				fmt.Println("You need to start the server first by choosing the menu option:", actnStartServer)
				continue
			}

			app.doAcceptConnection()

		case actnStartServer:
			go app.startServer()
			time.Sleep(time.Second / 5)
			app.cfg.RunServer = true
			app.cfg.Save()

		case actnStopServer:
			app.svrStop()
			app.cfg.RunServer = false
			app.cfg.Save()

		case actnAddAuthorizedKey:
			app.addAuthorizedKey()

		case actnShowPubKey:
			fmt.Println("Paste this public key in another eztunnel instance to authorize this instance")
			fmt.Println(app.cfg.PublicKey)

		// case actnGenerateHostKey:
		// case actnGeneratePubKey:

		case actnDumpConfig:
			fmt.Printf("%+v\n", app.cfg)

		case actnDumpConfigFile:
			f, err := os.Open("./" + app.cfg.Filename)
			if err != nil {
				panic(err)
			}

			_, err = io.Copy(os.Stdout, f)
			if err != nil {
				panic(err)
			}

		case actnQuit:
			return

		}
	}
}

func (app *appMgr) doAcceptConnection() {
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

func (app *appMgr) doConnect() {
	var hn, lport, rport string

	for hn == "" {
		survey.AskOne(&survey.Input{
			Message: "Enter the hostname (and port) to connect to:",
			Help:    "192.168.1.1:4443",
			Default: "",
		}, &hn)
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
}

func (app *appMgr) startServer() {
	defer func() { app.svrIsRunning = false }()
	var ctx context.Context
	ctx, app.svrStop = context.WithCancel(context.Background())
	app.svr = eztunnel.NewServer(app.cfg)
	log.Println("starting server on port", app.cfg.ListenPort)
	app.svrIsRunning = true
	app.svr.ListenAndServe(ctx)
}

func (app *appMgr) stopTunnels() {
	app.tunMgr.CloseAll()
}

func (app *appMgr) startTunnels() {
	app.tunMgr = eztunnel.NewTunnelManager(app.cfg)

	app.tunMgr.OpenAll()
}

func (app *appMgr) addAuthorizedKey() {
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
