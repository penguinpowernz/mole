package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	eztunnel "github.com/penguinpowernz/eztunnel/pkg"
)

func main() {
	var cfgFile string
	var daemonize bool
	flag.StringVar(&cfgFile, "c", "config.yml", "the config file to use")
	flag.BoolVar(&daemonize, "d", false, "daemonize using the given config")
	flag.Parse()

	if err := generateConfigIfNeeded(cfgFile); err != nil {
		panic(err)
	}

	cfg, err := eztunnel.LoadConfig(cfgFile)
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}

	app := appMgr{cfg: cfg}

	if cfg.RunServer {
		go app.startServer()
		time.Sleep(time.Second / 5)
	}

	go app.startTunnels()
	defer app.stopTunnels()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !daemonize {
		fmt.Println("Push enter to show the menu, otherwise logs will be printed")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		app.showMenu()
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
	if app.svrIsRunning {
		app.svrStop()
	}
}
