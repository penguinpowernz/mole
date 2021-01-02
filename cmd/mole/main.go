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
	"github.com/penguinpowernz/eztunnel/pkg/tunnel"
)

func main() {

	var addr, remote, local, keyfile string
	flag.StringVar(&addr, "a", "127.0.0.1:22", "the address to connect to")
	flag.StringVar(&remote, "r", "", "the remote port")
	flag.StringVar(&local, "l", "", "the local port")
	flag.StringVar(&keyfile, "i", os.Getenv("HOME")+"/.ssh/id_rsa", "identity file (private key)")

	flag.Parse()

	data, err := ioutil.ReadFile(keyfile)
	if err != nil {
		panic(err)
	}
	log.Printf("found keyfile at: %s", keyfile)

	if remote[0] == ':' || (remote[0] != ':' && !strings.Contains(remote, ":")) {
		remote = "localhost:" + remote
	}

	if local[0] == ':' || (local[0] != ':' && !strings.Contains(local, ":")) {
		local = "localhost:" + local
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
	events.On("client.connected", func(cl *tunnel.Client) {
		log.Println("they say its connected")
		fmt.Println(cl)
	})

	pool := tunnel.NewConnPool(ctx, events)

	log.Println("opening tunnel to", addr, remote, "for local port", local)
	tun, err := tunnel.NewTunnelFromPool(pool, addr, remote, local, string(data))
	if err != nil {
		panic(err)
	}

	if err := tun.Open(); err != nil {
		panic(err)
	}
	log.Println("opened tunnel")
	defer tun.Close()
	tun.Listen(events)

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
