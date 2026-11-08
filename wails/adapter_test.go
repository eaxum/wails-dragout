package wails

import (
	"context"
	"errors"
	"testing"

	dragout "github.com/eaxum/wails-dragout"
)

func TestStartWithOptionsStopsBeforeNativeStartWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dispatched := false
	effect, err := startWithOptions(ctx, nil, nil, dragout.Options{}, func(callback func()) {
		dispatched = true
		callback()
	})
	if effect != dragout.EffectNone || !errors.Is(err, context.Canceled) {
		t.Fatalf("effect=%d err=%v", effect, err)
	}
	if !dispatched {
		t.Fatal("UI callback was not dispatched")
	}
}
