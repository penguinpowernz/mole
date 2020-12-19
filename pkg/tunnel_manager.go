package eztunnel

import "context"

func NewTunnelManager(cfg *Config) *TunnelManager {
	tm := new(TunnelManager)
	tm.cfg = cfg
	return tm
}

type TunnelManager struct {
	cfg *Config
	cls []*Client
}

func (mgr *TunnelManager) Tunnels() (tuns []*Tunnel) {
	for _, cl := range mgr.cls {
		for _, tun := range cl.tuns {
			tuns = append(tuns, tun)
		}
	}
	return tuns
}

func (mgr *TunnelManager) Start(ctx context.Context) {
	<-ctx.Done()
	for _, cl := range mgr.cls {
		cl.Close()
	}
}

func (mgr *TunnelManager) CloseAll() {
	mgr.CloseTunnels(mgr.cfg.Tunnels...)
}

func (mgr *TunnelManager) OpenAll() {
	mgr.OpenTunnels(mgr.cfg.Tunnels...)
}

func (mgr *TunnelManager) OpenTunnels(tuns ...Tunnel) {
	for _, tun := range tuns {
		if !mgr.HaveClient(tun.Address) {
			mgr.AddClient(tun.Address)
		}

		for _, cl := range mgr.cls {
			cl.OpenTunnel(tun)
		}
	}
}

func (mgr *TunnelManager) CloseTunnels(tuns ...Tunnel) {
	for _, tun := range tuns {
		if !mgr.HaveClient(tun.Address) {
			mgr.AddClient(tun.Address)
		}

		for _, cl := range mgr.cls {
			cl.CloseTunnel(tun)
		}
	}
}

func (mgr *TunnelManager) AddClient(addr string) error {
	cl, err := NewClient(addr, mgr.cfg.PrivateKey)
	if err != nil {
		return err
	}

	err = cl.Connect()
	if err != nil {
		return err
	}

	mgr.cls = append(mgr.cls, cl)
	return nil
}

func (mgr *TunnelManager) HaveClient(addr string) bool {
	for _, cl := range mgr.cls {
		if cl.addr == addr {
			return true
		}
	}
	return false
}
