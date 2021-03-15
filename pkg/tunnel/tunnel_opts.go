package tunnel

import "sync"

func NewTunnelFromOpts(opts ...Option) (*Tunnel, error) {
	t := &Tunnel{}
	for _, opt := range opts {
		if err := opt(t); err != nil {
			return t, err
		}
	}

	return t, nil
}

type Option func(*Tunnel) error

func Local(bind string) Option {
	return func(tun *Tunnel) error {
		tun.Local = bind
		return nil
	}
}

func Remote(bind string) Option {
	return func(tun *Tunnel) error {
		tun.Remote = bind
		return nil
	}
}

func Reverse() Option {
	return func(tun *Tunnel) error {
		tun.Reverse = true
		return nil
	}
}

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
