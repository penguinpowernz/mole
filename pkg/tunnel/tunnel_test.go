package tunnel

import (
	"fmt"
	"testing"
	// . "github.com/smartystreets/goconvey/convey"
)

// func TestSomething(t *testing.T) {
// 	Convey("", t, func() {

// 	})
// }

func TestPortNormalization(t *testing.T) {
	tun := &Tunnel{}
	tun.Remote = "0.0.0.0:6222"
	tun.Local = "localhost:6555"
	tun.normalizePorts()

	if tun.Remote != "0.0.0.0:6222" {
		t.Fail()
	}

	if tun.Local != "localhost:6555" {
		t.Fail()
	}

	tun.Remote = "6222"
	tun.Local = "6555"
	tun.normalizePorts()

	if tun.Remote != "localhost:6222" {
		t.Fail()
	}

	if tun.Local != "localhost:6555" {
		t.Fail()
	}

	tun.Remote = ":6222"
	tun.Local = ":6555"
	tun.normalizePorts()
	fmt.Println(tun.Remote, tun.Local)
	if tun.Remote != "localhost:6222" {
		t.Fail()
	}

	if tun.Local != "localhost:6555" {
		t.Fail()
	}
}
