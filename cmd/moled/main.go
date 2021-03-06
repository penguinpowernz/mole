package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AlexanderGrom/go-event"
	"github.com/penguinpowernz/mole/internal/app"
	"github.com/penguinpowernz/mole/internal/util"
	"github.com/penguinpowernz/mole/pkg/tunnel/server"
)

var svr *server.Server

func main() {
	var cfgFile, generateConfig, port string
	var interactiveAccept, interactiveUDS bool
	flag.StringVar(&generateConfig, "g", "", "generate a new config file to the given location")
	flag.StringVar(&cfgFile, "c", "", "the config file to use")
	flag.StringVar(&port, "p", "", "the port to serve the server on")
	flag.BoolVar(&interactiveAccept, "i", false, "interactively accept public keys (useful for setting up)")
	flag.BoolVar(&interactiveUDS, "I", false, "don't run the server, just listen for public key requests")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !interactiveUDS {
		if generateConfig != "" {
			tryToGenerateConfig(generateConfig)
		}

		cfg := loadConfig(cfgFile)

		if port = normalizePort(port); port != "" {
			cfg.ListenPort = port
		}

		if !cfg.RunServer {
			log.Fatal("configured to not run server, nothing to do...")
		}

		events := event.New()
		logEvents(events)

		svr := server.NewServer(cfg, events)
		go runServer(ctx, cfg, svr)

		if interactiveAccept {
			server.InteractivelyAcceptPublicKeys(svr, cfg)
			return
		}
	}
	if interactiveUDS {
		// run the interactive public key authorizer, listen for requests
		// from an already running moled process
		go func() {
			err := app.UDSAuthServer(ctx)
			if err != nil {
				log.Println(err)
				cancel()
			}
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

func runServer(ctx context.Context, cfg *server.Config, svr *server.Server) {
	for {
		log.Println("starting server on port", cfg.ListenPort)
		svr.ListenAndServe(ctx)
		log.Println("server stopped")

		// don't loop if the ctx was done
		if ctx.Err() == nil {
			return
		}

		time.Sleep(time.Second)
	}
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return false
	}
	return true
}

// try to load the config from the specified file, if none was specified
// then try the config search paths, if none was found then generate to
// the first file in the config search paths
func loadConfig(specifiedFilename string) *server.Config {
	if specifiedFilename == "" {
		fn, found := util.FindConfig()
		if !found {
			fn = util.ConfigFiles[0]
			log.Println("config file not found, generating one at", fn)
			if err := server.GenerateConfigIfNeeded(fn); err != nil {
				panic(err)
			}
		} else {
			specifiedFilename = fn
		}
	}

	cfg, err := server.LoadConfig(specifiedFilename)
	if err != nil {
		log.Fatal("Failed to load config from", specifiedFilename, "-", err)
	}

	log.Println("loaded config file from", specifiedFilename)
	return cfg
}

// try to ensure the port number is in the correct format for net.Dialer
func normalizePort(port string) string {
	if port != "" && !strings.HasPrefix(port, ":") && !strings.Contains(port, ":") {
		port = ":" + port
	}

	return port
}

func tryToGenerateConfig(specifiedFilename string) {
	if fileExists(specifiedFilename) {
		fmt.Println("ERROR: file already exists at", specifiedFilename)
		os.Exit(1)
	}
	cfg := server.GenerateConfig()
	cfg.Filename = specifiedFilename
	if err := cfg.Save(); err != nil {
		panic(err)
	}
	fmt.Println("New config file generated and saved to", specifiedFilename)
	os.Exit(0)
}

// watch the events and log any relevant ones
func logEvents(events event.Dispatcher) {
	events.On("error", func(err error) error {
		log.Println("ERROR:", err)
		return nil
	})
	events.On("log", func(msg string) error {
		log.Println(msg)
		return nil
	})
}
