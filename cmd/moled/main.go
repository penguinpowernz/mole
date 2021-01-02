package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AlexanderGrom/go-event"
	"github.com/penguinpowernz/eztunnel/internal/util"
	"github.com/penguinpowernz/eztunnel/pkg/tunnel/server"
)

func main() {
	var cfgFile, port string
	flag.StringVar(&cfgFile, "c", "config.yml", "the config file to use")
	flag.StringVar(&port, "p", "", "the port to serve the server on")
	flag.Parse()

	if port != "" && !strings.HasPrefix(port, ":") && !strings.Contains(port, ":") {
		port = ":" + port
	}

	if err := util.GenerateConfigIfNeeded(cfgFile); err != nil {
		panic(err)
	}

	cfg, err := server.LoadConfig(cfgFile)
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}

	if port != "" {
		cfg.ListenPort = port
	}

	if !cfg.RunServer {
		log.Fatal("not running server, nothing to do...")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := event.New()
	events.On("error", func(err error) error {
		log.Println("ERROR:", err)
		return nil
	})
	events.On("log", func(msg string) error {
		log.Println(msg)
		return nil
	})

	go RunServer(ctx, cfg, events)

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

func RunServer(ctx context.Context, cfg *server.Config, events event.Dispatcher) {
	svr := server.NewServer(cfg, events)
	log.Println("starting server on port", cfg.ListenPort)
	svr.ListenAndServe(ctx)
}
