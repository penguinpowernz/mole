package app

import (
	"context"
	"log"

	eztunnel "github.com/penguinpowernz/eztunnel/pkg"
)

func (app *App) StartServer() {
	defer func() { app.svrIsRunning = false }()
	var ctx context.Context
	ctx, app.svrStop = context.WithCancel(context.Background())
	app.svr = eztunnel.NewServer(app.cfg, app.events)
	log.Println("starting server on port", app.cfg.ListenPort)
	app.svrIsRunning = true
	app.events.Go("log", "starting server listening on port ", app.cfg.ListenPort)
	app.svr.ListenAndServe(ctx)
	app.events.Go("log", "server stopped running")
}

func (app *App) StopTunnels() {
	app.tunMgr.CloseAll()
}

func (app *App) StartTunnels() {
	app.tunMgr = eztunnel.NewTunnelManager(app.cfg, app.events)
	app.tunMgr.OpenAll()
}
