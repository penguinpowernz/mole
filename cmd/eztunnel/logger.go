package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/AlexanderGrom/go-event"
)

type cachedLogger struct {
	events  event.Dispatcher
	enabled bool
	caching bool
	cache   []string
}

func (cl cachedLogger) Listen() {
	cl.events.On("log", func(msg string, args ...interface{}) error {
		if cl.enabled && cl.caching {
			buf := &strings.Builder{}
			log.New(buf, "", log.LstdFlags).Printf(msg, args...)
			cl.cache = append(cl.cache, buf.String())
			return nil
		}

		log.Println(msg)
		return nil
	})

	cl.events.On("error", func(err error) error {
		if cl.enabled && cl.caching {
			buf := &strings.Builder{}
			log.New(buf, "", log.LstdFlags).Print("ERROR:", err)
			cl.cache = append(cl.cache, buf.String())
			return nil
		}

		log.Println("ERROR:", err)
		return nil
	})

	cl.events.On("log.cache", func() error {
		cl.StartCaching()
		return nil
	})

	cl.events.On("log.uncache", func() error {
		cl.StopCaching()
		return nil
	})
}

func (cl cachedLogger) StartCaching() {
	cl.caching = true
}

func (cl cachedLogger) StopCaching() {
	for _, msg := range cl.cache {
		fmt.Println(msg)
	}
	cl.caching = false
	cl.cache = []string{}
}
