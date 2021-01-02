package eztunnel

import (
	"context"
	"log"

	"github.com/AlexanderGrom/go-event"
)

func NewTunnelManager(cfg *Config, events event.Dispatcher) *TunnelManager {
	tm := new(TunnelManager)
	tm.cfg = cfg
	tm.events = events
	return tm
}

type TunnelManager struct {
	cfg    *Config
	events event.Dispatcher
	cls    []*Client
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
	mgr.events.Go("log", "opening all tunnels")
	mgr.OpenTunnels(mgr.cfg.Tunnels...)
}

func (mgr *TunnelManager) OpenTunnels(tuns ...Tunnel) {
	for _, tun := range tuns {
		if !mgr.HaveClient(tun.Address) {
			// log.Println("adding new client", tun.Address)
			mgr.events.Go("log", "connecting to new client @ "+tun.Address)
			if err := mgr.AddClient(tun.Address); err != nil {
				mgr.events.Go("error", err)
			}
		}

		for _, cl := range mgr.cls {
			if err := cl.OpenTunnel(tun); err != nil {
				log.Printf("failed to open tunnel %s : %s: %s", tun.Address, tun.Name(), err)
				mgr.events.Go("log", "failed to open tunnel %s : %s: %s", tun.Address, tun.Name(), err)
			} else {
				log.Printf("connected tunnel %s : %s", tun.Address, tun.Name())
				mgr.events.Go("log", "connected tunnel %s : %s", tun.Address, tun.Name())
			}
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
