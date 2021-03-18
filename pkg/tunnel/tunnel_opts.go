package tunnel

import (
	"sync"

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

// BuildTunnelswill build a collection of tunnels from a config
func BuildTunnels(cfg Config) []*Tunnel {
	tuns := []*Tunnel{}
	for _, t := range cfg.Tunnels {
		if !t.Enabled {
			continue
		}

		t := &Tunnel{
			Address: t.Address,
			Local:   t.Local,
			Remote:  t.Remote,
			Reverse: t.Reverse,
			Enabled: true,
			mu:      new(sync.Mutex),
		}

		tuns = append(tuns, t)
	}

	return tuns
}
