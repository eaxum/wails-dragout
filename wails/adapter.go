package wails

import (
	"context"

	"github.com/eaxum/wails-dragout"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Start waits on the calling goroutine while native drag runs on the UI thread.
func Start(ctx context.Context, window *application.WebviewWindow, paths []string) (dragout.Result, error) {
	type outcome struct {
		result dragout.Result
		err    error
	}
	completed := make(chan outcome, 1)
	application.InvokeAsync(func() {
		if err := ctx.Err(); err != nil {
			completed <- outcome{err: err}
			return
		}
		dragout.Begin(window.NativeWindow(), paths, func(result dragout.Result, err error) {
			completed <- outcome{result, err}
		})
	})
	// Cancellation must not free an active native source or allow an overlapping session.
	result := <-completed
	return result.result, result.err
}

// StartWithOptions dispatches an effect-aware drag to the Wails UI thread and
// waits until the native operation completes.
func StartWithOptions(ctx context.Context, window *application.WebviewWindow, paths []string, options dragout.Options) (dragout.Effect, error) {
	return startWithOptions(ctx, window, paths, options, application.InvokeAsync)
}

func startWithOptions(ctx context.Context, window *application.WebviewWindow, paths []string, options dragout.Options, invoke func(func())) (dragout.Effect, error) {
	type outcome struct {
		effect dragout.Effect
		err    error
	}
	completed := make(chan outcome, 1)
	invoke(func() {
		if err := ctx.Err(); err != nil {
			completed <- outcome{err: err}
			return
		}
		dragout.BeginWithOptions(window.NativeWindow(), paths, options, func(effect dragout.Effect, err error) {
			completed <- outcome{effect, err}
		})
	})
	// Keep the native source alive and the caller's session guard held until the drag ends.
	result := <-completed
	return result.effect, result.err
}
