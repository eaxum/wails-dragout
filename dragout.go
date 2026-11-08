package dragout

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

type Result string

const (
	Dropped   Result = "dropped"
	Cancelled Result = "cancelled"
)

// Effect is the operation requested from or reported by a native drag target.
type Effect uint32

const EffectNone Effect = 0

const (
	EffectCopy Effect = 1 << iota
	EffectMove
	EffectLink
)

// Effects is a bit set of drag operations.
type Effects uint32

const (
	EffectSetCopy Effects = Effects(EffectCopy)
	EffectSetMove Effects = Effects(EffectMove)
	EffectSetLink Effects = Effects(EffectLink)
	EffectSetAll          = EffectSetCopy | EffectSetMove | EffectSetLink
)

// Options controls which effects a drag may perform and which one it prefers.
type Options struct {
	AllowedEffects  Effects
	PreferredEffect Effect
}

var (
	ErrUnsupported          = errors.New("native file dragging is not available on this platform")
	ErrUnsupportedEffect    = errors.New("requested drag effect is not supported on this platform")
	ErrInvalidEffectOptions = errors.New("invalid drag effect options")
)

// SupportedEffects reports the effects implemented by the current build.
func SupportedEffects() Effects {
	if !Available {
		return 0
	}
	switch runtime.GOOS {
	case "windows":
		return EffectSetAll
	case "darwin", "linux":
		return EffectSetCopy
	default:
		return 0
	}
}

// BeginWithOptions starts a native drag using the selected effects. Call it on
// the platform UI thread. EffectNone means the drag was cancelled or the
// accepted drop did not provide a reliable effect; it is never confirmation
// that source files should be deleted.
func BeginWithOptions(window unsafe.Pointer, paths []string, options Options, complete func(Effect, error)) {
	if err := validateOptions(options, EffectSetAll); err != nil {
		complete(EffectNone, err)
		return
	}
	if !Available {
		complete(EffectNone, ErrUnsupported)
		return
	}
	if err := validateOptions(options, SupportedEffects()); err != nil {
		complete(EffectNone, err)
		return
	}
	beginWithOptions(window, paths, options, complete)
}

// Begin starts a copy-only native drag and preserves the legacy completion result.
func Begin(window unsafe.Pointer, paths []string, complete func(Result, error)) {
	BeginWithOptions(window, paths, Options{
		AllowedEffects:  EffectSetCopy,
		PreferredEffect: EffectCopy,
	}, func(effect Effect, err error) {
		if err != nil {
			complete("", err)
			return
		}
		if effect == EffectNone {
			complete(Cancelled, nil)
			return
		}
		complete(Dropped, nil)
	})
}

func validateOptions(options Options, supported Effects) error {
	if options.AllowedEffects == 0 {
		return fmt.Errorf("%w: at least one allowed effect is required", ErrInvalidEffectOptions)
	}
	if unsupported := options.AllowedEffects &^ supported; unsupported != 0 {
		return fmt.Errorf("%w: %b", ErrUnsupportedEffect, unsupported)
	}
	preferred := Effects(options.PreferredEffect)
	if preferred == 0 || preferred&(preferred-1) != 0 || options.AllowedEffects&preferred == 0 {
		return fmt.Errorf("%w: preferred effect must be one allowed effect", ErrInvalidEffectOptions)
	}
	return nil
}

func resolveEffect(allowed Effects, dropped bool, logicalEffect Effect, logicalAvailable bool,
	performedEffect Effect, performedAvailable bool, returnedEffect Effect) Effect {
	if !dropped {
		return EffectNone
	}
	if logicalAvailable {
		return confirmedEffect(logicalEffect, allowed)
	}
	if performedAvailable {
		return confirmedEffect(performedEffect, allowed)
	}
	return confirmedEffect(returnedEffect, allowed)
}

func confirmedEffect(effect Effect, allowed Effects) Effect {
	set := Effects(effect)
	if set == 0 || set&(set-1) != 0 || set&allowed == 0 || set&^EffectSetAll != 0 {
		return EffectNone
	}
	return effect
}

func effectFromCopyOnlyResult(result Result) Effect {
	if result == Dropped {
		return EffectCopy
	}
	return EffectNone
}
