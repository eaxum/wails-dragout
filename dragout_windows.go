//go:build windows && cgo

package dragout

/*
#cgo LDFLAGS: -lole32 -lshell32 -luuid -luser32 -lgdi32
#include "dragout_windows.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unsafe"
)

const Available = true

const (
	nativeDropped   = 0x00040100
	nativeCancelled = 0x00040101
)

func startWithOptions(window unsafe.Pointer, paths []string, options Options) (Result, Effect, error) {
	encoded, err := encodePaths(paths)
	if err != nil {
		return "", EffectNone, err
	}
	var returnedEffect C.uint32_t
	var logicalEffect C.uint32_t
	var hasLogicalEffect C.int32_t
	var performedEffect C.uint32_t
	var hasPerformedEffect C.int32_t
	result := uint32(C.clustta_drag_files(
		C.uintptr_t(uintptr(window)),
		(*C.uint16_t)(unsafe.Pointer(&encoded[0])),
		C.size_t(len(paths)),
		C.uint32_t(options.AllowedEffects),
		C.uint32_t(options.PreferredEffect),
		&returnedEffect,
		&logicalEffect,
		&hasLogicalEffect,
		&performedEffect,
		&hasPerformedEffect,
	))
	switch result {
	case nativeDropped:
		returned := Effect(returnedEffect)
		return legacyWindowsResult(true, returned), resolveEffect(
			options.AllowedEffects,
			true,
			Effect(logicalEffect),
			hasLogicalEffect != 0,
			Effect(performedEffect),
			hasPerformedEffect != 0,
			returned&Effect(EffectSetAll),
		), nil
	case nativeCancelled:
		return Cancelled, EffectNone, nil
	default:
		return "", EffectNone, fmt.Errorf("Windows could not drag the selected files (HRESULT 0x%08X)", result)
	}
}

func encodePaths(paths []string) ([]uint16, error) {
	if len(paths) == 0 {
		return nil, errors.New("select at least one local file")
	}
	var encoded []uint16
	for _, path := range paths {
		if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
			return nil, errors.New("drag paths must be absolute and contain no null characters")
		}
		encoded = append(encoded, utf16.Encode([]rune(path))...)
		encoded = append(encoded, 0)
	}
	return append(encoded, 0), nil
}

func probeFiles(paths []string) error {
	encoded, err := encodePaths(paths)
	if err != nil {
		return err
	}
	result := uint32(C.clustta_probe_drag_files((*C.uint16_t)(unsafe.Pointer(&encoded[0])), C.size_t(len(paths))))
	if result != 0 {
		return fmt.Errorf("shell data object failed: 0x%08X", result)
	}
	return nil
}

func probePreferredEffect(paths []string, effect Effect) error {
	encoded, err := encodePaths(paths)
	if err != nil {
		return err
	}
	result := uint32(C.clustta_probe_preferred_effect(
		(*C.uint16_t)(unsafe.Pointer(&encoded[0])),
		C.size_t(len(paths)),
		C.uint32_t(effect),
	))
	if result != 0 {
		return fmt.Errorf("Shell data object did not preserve preferred effect %d (HRESULT 0x%08X)", effect, result)
	}
	return nil
}

func probeDropEffectFormats(paths []string, logical Effect, hasLogical bool, performed Effect, hasPerformed bool) error {
	encoded, err := encodePaths(paths)
	if err != nil {
		return err
	}
	result := uint32(C.clustta_probe_drop_effect_formats(
		(*C.uint16_t)(unsafe.Pointer(&encoded[0])),
		C.size_t(len(paths)),
		C.uint32_t(logical),
		boolToCInt32(hasLogical),
		C.uint32_t(performed),
		boolToCInt32(hasPerformed),
	))
	if result != 0 {
		return fmt.Errorf("Shell data object did not preserve completion effects (HRESULT 0x%08X)", result)
	}
	return nil
}

func boolToCInt32(value bool) C.int32_t {
	if value {
		return 1
	}
	return 0
}
