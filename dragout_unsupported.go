//go:build (!windows && !darwin && !linux) || !cgo

package dragout

import "unsafe"

const Available = false

func start(window unsafe.Pointer, paths []string) (Result, error) {
	return "", ErrUnsupported
}
