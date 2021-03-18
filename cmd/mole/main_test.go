package main

import (
	"fmt"
	"testing"
)

func TestLPFLocal(t *testing.T) {
	l, r := breakApartLPF("1234:localhost:4568")
	if l != "localhost:4568" {
		t.Fail()
	}
	if r != "localhost:1234" {
		t.Fail()
	}
}

func TestLPFLocalSimple(t *testing.T) {
	l, r := breakApartLPF("1234:4568")
	if l != "localhost:4568" {
		t.Fail()
	}
	if r != "localhost:1234" {
		t.Fail()
	}
}

func TestLPFRemote(t *testing.T) {
	l, r := breakApartLPF("0.0.0.0:1234:localhost:4568")
	fmt.Println(l, r)
	if l != "localhost:4568" {
		t.Fail()
	}
	if r != "0.0.0.0:1234" {
		t.Fail()
	}
}

func TestLPFRemoteExtraColon(t *testing.T) {
	l, r := breakApartLPF(":1234:localhost:4568")
	if l != "localhost:4568" {
		t.Fail()
	}
	if r != "localhost:1234" {
		t.Fail()
	}
}
