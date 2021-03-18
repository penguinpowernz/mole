package sshutil

import "strings"

// ParsePortForwardDefinition will parse an SSH port forward definition
// and return a local port and remote port, adding 127.0.0.1 to ambiguous
// port. Check the following examples:
//
//     Definition               Local          Remote
//     11:localhost:22          localhost:22   127.0.0.1:11
//     0.0.0.0:11:localhost:22  localhost:22   0.0.0.0:11
//     11:22                    127.0.0.1:22   127.0.0.1:11
//
func ParsePortForwardDefinition(pf string) (string, string) {
	bits := strings.Split(pf, ":")

	var r, l string

	switch strings.Count(pf, ":") {
	case 3: // 0.0.0.0:1234:localhost:1234
		r = strings.Join(bits[0:2], ":")
		l = strings.Join(bits[2:], ":")
		if pf[0] == ':' {
			r = "127.0.0.1" + r
		}

	case 2: // 1234:localhost:5678
		r = "127.0.0.1:" + bits[0]
		l = strings.Join(bits[1:], ":")
	case 1: // "1234:5678"
		r = "127.0.0.1:" + bits[0]
		l = "127.0.0.1:" + bits[1]
	}

	return l, r
}
