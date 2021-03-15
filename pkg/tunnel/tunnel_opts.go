package tunnel

import "sync"
func BuildTunnels(pool Pool, cfg Config) ([]*Tunnel, error) {
	tuns := []*Tunnel{}
	for _, t := range cfg.Tunnels {
		if !t.Enabled {
			continue
		}

		cl, err := pool.GetClient(t.Address, cfg.KeyForAddress(t.Address))
		if err != nil {
			return tuns, err
		}

		t := &Tunnel{
			Address: t.Address,
			Local:   t.Local,
			Remote:  t.Remote,
			Reverse: t.Reverse,
			dialer:  cl.dialerFunc(),
			Enabled: true,
			mu:      new(sync.Mutex),
		}

		tuns = append(tuns, t)
	}

	return tuns, nil
}
