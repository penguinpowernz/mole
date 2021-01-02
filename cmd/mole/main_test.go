package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBreakLPFApart(t *testing.T) {
	Convey("given some LPFs", t, func() {
		Convey("they should be broken apart properly", func() {
			l, r := breakApartLPF("1234:localhost:4568")
			So(r, ShouldEqual, "localhost:4568")
			So(l, ShouldEqual, "1234")

			l, r = breakApartLPF("1234:4568")
			So(r, ShouldEqual, "4568")
			So(l, ShouldEqual, "1234")
		})
	})
}
