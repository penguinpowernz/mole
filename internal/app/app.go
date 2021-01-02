package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AlecAivazis/survey"
	"github.com/AlecAivazis/survey/terminal"
	"github.com/AlexanderGrom/go-event"
	"github.com/penguinpowernz/eztunnel/internal/util"
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

type App struct {
	cfg    *eztunnel.Config
	events event.Dispatcher
	ctx    context.Context

	svr          *eztunnel.Server
	svrStop      func()
	svrIsRunning bool

	tunMgr *eztunnel.TunnelManager
}

func New(ctx context.Context, cfg *eztunnel.Config, events event.Dispatcher) *App {
	app := &App{
		cfg:    cfg,
		events: events,
		ctx:    ctx,
	}

	app.tunMgr = new(eztunnel.TunnelManager)

	go func() {
		<-ctx.Done()
		app.cleanup()
	}()

	return app
}

func (app *App) ShowMenu() {
	for {
		var actn string
		serverToggle := actnStartServer
		if app.svrIsRunning {
			serverToggle = actnStopServer
		}

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

		if app.ctx.Err() != nil {
			return
		}

		switch actn {

		case actnWatchLogs:
			util.PrintLogsUntilEnter()

		case actnConnect:
			app.doConnect()

		case actnAcceptConnection:
			if !app.svrIsRunning {
				fmt.Println("You need to start the server first by choosing the menu option:", actnStartServer)
				continue
			}

			app.doAcceptConnection()

		case actnStartServer:
			go app.StartServer()
			time.Sleep(time.Second / 5)
			app.cfg.RunServer = true
			app.cfg.Save()

		case actnStopServer:
			app.svrStop()
			app.cfg.RunServer = false
			app.cfg.Save()

		case actnAddAuthorizedKey:
			app.doAddAuthorizedKey()

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

func (app *App) cleanup() {
	if app.svrIsRunning {
		app.svrStop()
	}
	app.tunMgr.CloseAll()
}
