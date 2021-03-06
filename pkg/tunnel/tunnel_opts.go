package tunnel

import (
	"github.com/penguinpowernz/mole/pkg/sshutil"
)

// Option is an option for the tunnel
type Option func(*Tunnel) error

// Local will set the local port binding for the tunnel
func Local(bind string) Option {
	return func(tun *Tunnel) error {
		tun.Local = bind
		return nil
	}
}

// Remote will set the remote port binding for the tunnel
func Remote(bind string) Option {
	return func(tun *Tunnel) error {
		tun.Remote = bind
		return nil
	}
}

// Reverse will set a tunnel as being reverse
func Reverse() Option {
	return func(tun *Tunnel) error {
		tun.Reverse = true
		return nil
	}
}

// PFD will set the tunnel ports up using the given SSH port forward definition
func PFD(def string) Option {
	return func(tun *Tunnel) error {
		tun.Local, tun.Remote = sshutil.ParsePortForwardDefinition(def)
		return nil
	}
}
