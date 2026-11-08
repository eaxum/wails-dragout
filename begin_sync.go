//go:build (!darwin && !linux) || !cgo

package dragout

import "unsafe"

// beginWithOptions delivers completion on the UI thread after the synchronous backend returns.
func beginWithOptions(window unsafe.Pointer, paths []string, options Options, complete func(Result, Effect, error)) {
	result, effect, err := startWithOptions(window, paths, options)
	complete(result, effect, err)
}
