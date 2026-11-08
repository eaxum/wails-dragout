//go:build (!windows && !darwin && !linux) || !cgo

package dragout

import "unsafe"

const Available = false

func startWithOptions(window unsafe.Pointer, paths []string, options Options) (Result, Effect, error) {
	return "", EffectNone, ErrUnsupported
}
