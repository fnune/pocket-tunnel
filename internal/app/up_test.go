package app

import (
	"context"
	"strings"
	"testing"
)

func TestUpRunsTheTunnelWhileHoldingTheLock(t *testing.T) {
	lock := &fakeLock{}
	runner := &fakeRunner{}
	if err := NewUp(&fakeStore{cfg: complete()}, lock, runner).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runner.ran == nil || !lock.acquired || !lock.released {
		t.Errorf("ran = %v, acquired = %v, released = %v", runner.ran != nil, lock.acquired, lock.released)
	}
}

func TestASecondUpDoesNotStartAnotherTunnel(t *testing.T) {
	runner := &fakeRunner{}
	err := NewUp(&fakeStore{cfg: complete()}, &fakeLock{heldBy: 4242}, runner).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "4242") {
		t.Errorf("err = %v, want one naming the running PID", err)
	}
	if runner.ran != nil {
		t.Error("a second tunnel was started")
	}
}

func TestUpRefusesAnIncompleteConfig(t *testing.T) {
	runner := &fakeRunner{}
	lock := &fakeLock{}
	err := NewUp(&fakeStore{cfg: template()}, lock, runner).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Interface Address") {
		t.Errorf("err = %v, want the missing fields", err)
	}
	if runner.ran != nil || lock.acquired {
		t.Error("an incomplete config started the tunnel")
	}
}
