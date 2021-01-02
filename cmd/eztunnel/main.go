package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexanderGrom/go-event"
	"github.com/penguinpowernz/eztunnel/internal/app"
	"github.com/penguinpowernz/eztunnel/internal/util"
	eztunnel "github.com/penguinpowernz/eztunnel/pkg"
)

func main() {
	var cfgFile string
	var daemonize, noListen, noConnect bool
	flag.StringVar(&cfgFile, "c", "config.yml", "the config file to use")
	flag.BoolVar(&daemonize, "d", false, "daemonize using the given config")
	flag.BoolVar(&noListen, "no-listen", false, "override config file to not run server")
	flag.BoolVar(&noConnect, "no-connect", false, "override config file to not connect to any servers")
	flag.Parse()

	if err := util.GenerateConfigIfNeeded(cfgFile); err != nil {
		panic(err)
	}

	cfg, err := eztunnel.LoadConfig(cfgFile)
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := event.New()
	app := app.New(ctx, cfg, events)
	logr := cachedLogger{events: events, enabled: !daemonize}
	logr.Listen()

	if cfg.RunServer && !noListen {
		go app.StartServer()
		time.Sleep(time.Second / 5)
	}

	if cfg.Connect && !noConnect {
		go app.StartTunnels()
	}

	if !daemonize {
		util.PrintLogsUntilEnter()
		go func() {
			app.ShowMenu()
			cancel()
		}()
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		<-sigc
		cancel()
	}()

	<-ctx.Done()
}
