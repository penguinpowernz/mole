package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gliderlabs/ssh"
)

var moleSocket = "/var/run/moled.sock"

func UDSAuthRequest(ctx ssh.Context) (bool, error) {
	var allow bool
	conn, err := net.Dial("unix", moleSocket)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	data, err := json.Marshal([]string{ctx.User(), ctx.RemoteAddr().String()})
	if err != nil {
		return false, err
	}

	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return false, err
	}

	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil {
		return false, err
	}

	res := string(buf[0])
	if res == "y" {
		allow = true
	}

	return allow, nil
}

func UDSAuthServer(ctx context.Context) error {
	ln, err := net.Listen("unix", moleSocket)
	if err != nil {
		return err
	}
	defer ln.Close()

	go func() {
		for {
			if ctx.Err() != nil {
				log.Println("context done")
				return
			}

			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)
				conn.Close()
				continue
			}
			defer conn.Close()

			buf := []byte{}
			_, err = conn.Read(buf)
			if err != nil {
				log.Println(err)
				conn.Close()
				continue
			}

			r := bufio.NewReader(conn)
			data, _, _ := r.ReadLine()

			pair := []string{}
			log.Println(string(data))
			if err := json.Unmarshal(data, &pair); err != nil {
				log.Println(err)
				conn.Close()
				continue
			}

			var allow bool
			survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Allow %s from %s to connect?", pair[0], pair[1]),
				Default: false,
			}, &allow)

			res := byte('n')
			if allow {
				res = byte('y')
			}

			_, err = conn.Write([]byte{res})
			if err != nil {
				log.Println(err)
				conn.Close()
				continue
			}

			conn.Close()
		}
	}()

	<-ctx.Done()
	ln.Close()
	return nil
}
