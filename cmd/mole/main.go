package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AlexanderGrom/go-event"
	"github.com/penguinpowernz/mole/internal/util"
	"github.com/penguinpowernz/mole/pkg/tunnel"
)

func main() {
	var addr, remote, local, generateConfig, localTunnel, keyfile, cfgFile string
	flag.StringVar(&addr, "a", "", "the address to connect to")
	flag.StringVar(&remote, "r", "", "the remote port")
	flag.StringVar(&local, "l", "", "the local port")
	flag.StringVar(&localTunnel, "L", "", "local port forward in SSH format")
	flag.StringVar(&keyfile, "i", "", "identity file (private key) to use, or override config with")
	flag.StringVar(&cfgFile, "c", "", "the config file to use")
	flag.StringVar(&generateConfig, "g", "", "generate a new config file to the given location")
	flag.Parse()

	if generateConfig != "" {
		tryToGenerateConfig(generateConfig)
	}

	if localTunnel != "" {
		local, remote = breakApartLPF(localTunnel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := event.New()
	logEvents(events)
	events.On("client.connected", func(cl *tunnel.Client) {
		log.Println("they say its connected")
		fmt.Println(cl)
	})

	pool := tunnel.NewConnPool(ctx, events)

	var cfg *tunnel.Config

	switch {
	case addr != "" && remote != "" && local != "":
		if keyfile == "" {
			keyfile = os.Getenv("HOME") + "/.ssh/id_rsa"
		}
		cfg = makeSingleTunnelConfig(addr, remote, local, keyfile)
	default:
		cfg = loadConfig(cfgFile, keyfile)
	}

	tuns, err := tunnel.NewTunnelsFromConfigAndPool(pool, *cfg)
	if err != nil {
		panic(err)
	}

	for _, tun := range tuns {
		log.Println("opening tunnel to", tun.Address, tun.Remote, "for local port", tun.Local)
		tun, err := tunnel.NewTunnelFromPool(pool, tun.Address, tun.Remote, tun.Local, cfg.PrivateKey)
		if err != nil {
			panic(err)
		}

		if err := tun.Open(); err != nil {
			panic(err)
		}
		log.Println("opened tunnel")
		defer tun.Close()
		tun.Listen(events)
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

	log.Println("waiting for quit signal")
	<-ctx.Done()
}

func makeSingleTunnelConfig(a, r, l, k string) *tunnel.Config {
	data, err := ioutil.ReadFile(k)
	if err != nil {
		panic(err)
	}
	log.Printf("found keyfile at: %s", k)

	return &tunnel.Config{
		PrivateKey: string(data),
		Tunnels: []tunnel.Tunnel{
			{Address: a, Local: l, Remote: r, Enabled: true},
		},
	}
}

func loadConfig(specifiedFilename, keyfile string) *tunnel.Config {
	if specifiedFilename == "" {
		fn, found := util.FindConfig()
		if !found {
			fn = util.ConfigFiles[0]
			log.Println("config file not found, generating one at", fn)
			if err := tunnel.GenerateConfigIfNeeded(fn); err != nil {
				panic(err)
			}
		} else {
			specifiedFilename = fn
		}
	}

	cfg, err := tunnel.LoadConfig(specifiedFilename)
	if err != nil {
		log.Fatal("Failed to load config from", specifiedFilename, "-", err)
	}

	if keyfile != "" {
		cfg.PrivateKey = privateKeyText(keyfile)
	}

	return cfg
}

func breakApartLPF(lpf string) (string, string) {
	bits := strings.Split(lpf, ":")
	return bits[0], strings.Join(bits[1:], ":")
}

func privateKeyText(keyfile string) string {
	data, err := ioutil.ReadFile(keyfile)
	if err != nil {
		panic(err)
	}
	log.Printf("found keyfile at: %s", keyfile)
	return string(data)
}

func tryToGenerateConfig(specifiedFilename string) {
	if fileExists(specifiedFilename) {
		fmt.Println("ERROR: file already exists at", specifiedFilename)
		os.Exit(1)
	}
	cfg := tunnel.GenerateConfig()
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

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return false
	}
	return true
}
