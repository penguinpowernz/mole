package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexanderGrom/go-event"
	"github.com/penguinpowernz/mole/internal/util"
	"github.com/penguinpowernz/mole/pkg/sshutil"
	"github.com/penguinpowernz/mole/pkg/tunnel"
)

func main() {
	var addr, remote, local, generateConfig, localTunnel, remoteTunnel, keyfile, cfgFile string
	var reverse bool
	flag.StringVar(&addr, "a", "", "the address to connect to")
	flag.StringVar(&remote, "r", "", "the remote port")
	flag.BoolVar(&reverse, "rr", false, "reverse port forward")
	flag.StringVar(&local, "l", "", "the local port")
	flag.StringVar(&localTunnel, "L", "", "local port forward in SSH format")
	flag.StringVar(&remoteTunnel, "R", "", "remote port forward in SSH format")
	flag.StringVar(&keyfile, "i", "", "identity file (private key) to use, or override config with")
	flag.StringVar(&cfgFile, "c", "", "the config file to use")
	flag.StringVar(&generateConfig, "g", "", "generate a new config file to the given location")
	flag.Parse()

	if generateConfig != "" {
		tryToGenerateConfig(generateConfig)
	}

	if localTunnel != "" {
		local, remote = sshutil.ParsePortForwardDefinition(localTunnel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := event.New()
	logEvents(events)
	events.On("client.connected", func(cl *tunnel.Client) {
		log.Println("they say its connected")
		fmt.Println(cl)
	})

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

	for _, cl := range cfg.Clients {
		go cl.OpenTunnels(ctx, events)
	}

	// USR1 will dump stats
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)
	go func() {
		for {
			<-sigusr1
			dumpStats(cfg.Tunnels().Open())
		}
	}()

	// any of these signals will do a graceful exit
	sigexit := make(chan os.Signal, 1)
	signal.Notify(sigexit,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		<-sigexit
		cancel()
	}()

	log.Println("waiting for quit signal")
	<-ctx.Done()
	time.Sleep(time.Second / 2)
}

func dumpStats(tuns []*tunnel.Tunnel) {
	for _, tun := range tuns {
		if tun.IsOpen {
			fmt.Println(tun)
		}
	}
}

func makeSingleTunnelConfig(a, r, l, k string) *tunnel.Config {
	data, err := ioutil.ReadFile(k)
	if err != nil {
		panic(err)
	}
	log.Printf("found keyfile at: %s", k)

	return &tunnel.Config{
		Clients: []*tunnel.Client{
			{
				Private: string(data),
				Address: a,
				Tunnels: []*tunnel.Tunnel{
					{Local: l, Remote: r},
				},
			},
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
		cfg.Clients = append(cfg.Clients, &tunnel.Client{Private: privateKeyText(keyfile), Address: "*"})
	}

	return cfg
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
	return err == nil
}
