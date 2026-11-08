package dragout

import (
	"errors"
	"runtime"
	"testing"
)

func TestValidateOptionsAcceptsPreferredAllowedEffect(t *testing.T) {
	options := Options{
		AllowedEffects:  EffectSetCopy | EffectSetMove,
		PreferredEffect: EffectMove,
	}
	if err := validateOptions(options, EffectSetCopy|EffectSetMove|EffectSetLink); err != nil {
		t.Fatalf("valid effects rejected: %v", err)
	}
}

func TestEffectsMatchOLEDropEffectValues(t *testing.T) {
	if EffectCopy != 1 || EffectMove != 2 || EffectLink != 4 {
		t.Fatalf("Copy/Move/Link=%d/%d/%d, want OLE values 1/2/4", EffectCopy, EffectMove, EffectLink)
	}
}

func TestCopyOnlyBackendResultMapsDropToCopy(t *testing.T) {
	if got := effectFromCopyOnlyResult(Dropped); got != EffectCopy {
		t.Fatalf("drop effect=%d, want %d", got, EffectCopy)
	}
	if got := effectFromCopyOnlyResult(Cancelled); got != EffectNone {
		t.Fatalf("cancel effect=%d, want %d", got, EffectNone)
	}
}

func TestLegacyWindowsResultPreservesCopyOnlyDropContract(t *testing.T) {
	tests := []struct {
		name           string
		dropped        bool
		returnedEffect Effect
		want           Result
	}{
		{name: "copy drop", dropped: true, returnedEffect: EffectCopy, want: Dropped},
		{name: "none effect", dropped: true, returnedEffect: EffectNone, want: Cancelled},
		{name: "move effect", dropped: true, returnedEffect: EffectMove, want: Cancelled},
		{name: "cancelled drag", returnedEffect: EffectCopy, want: Cancelled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := legacyWindowsResult(test.dropped, test.returnedEffect); got != test.want {
				t.Fatalf("legacyWindowsResult()=%q, want %q", got, test.want)
			}
		})
	}
}

func TestValidateOptionsRejectsEmptyAllowedEffects(t *testing.T) {
	options := Options{PreferredEffect: EffectCopy}
	if err := validateOptions(options, EffectSetCopy); err == nil {
		t.Fatal("empty allowed effects accepted")
	}
}

func TestValidateOptionsRejectsPreferredEffectNotAllowed(t *testing.T) {
	options := Options{AllowedEffects: EffectSetCopy, PreferredEffect: EffectMove}
	if err := validateOptions(options, EffectSetCopy|EffectSetMove); err == nil {
		t.Fatal("preferred effect outside allowed effects accepted")
	}
}

func TestValidateOptionsRejectsUnsupportedEffects(t *testing.T) {
	options := Options{AllowedEffects: EffectSetCopy | EffectSetMove, PreferredEffect: EffectCopy}
	if err := validateOptions(options, EffectSetCopy); err == nil {
		t.Fatal("unsupported effect was accepted")
	}
}

func TestValidateOptionsRejectsCombinedPreferredEffects(t *testing.T) {
	options := Options{AllowedEffects: EffectSetCopy | EffectSetMove, PreferredEffect: EffectCopy | EffectMove}
	if err := validateOptions(options, EffectSetCopy|EffectSetMove); err == nil {
		t.Fatal("multiple preferred effects were accepted")
	}
}

func TestSupportedEffectsMatchesAvailability(t *testing.T) {
	supported := SupportedEffects()
	want := Effects(0)
	if Available {
		switch runtime.GOOS {
		case "windows":
			want = EffectSetAll
		case "darwin", "linux":
			want = EffectSetCopy
		}
	}
	if supported != want {
		t.Fatalf("SupportedEffects()=%b, want %b (GOOS=%s, Available=%t)", supported, want, runtime.GOOS, Available)
	}
}

func TestResolveEffectUsesMostReliableReportedOutcome(t *testing.T) {
	tests := []struct {
		name               string
		allowed            Effects
		dropped            bool
		logicalEffect      Effect
		logicalAvailable   bool
		performedEffect    Effect
		performedAvailable bool
		returnedEffect     Effect
		want               Effect
	}{
		{
			name:               "logical effect identifies optimized move",
			allowed:            EffectSetAll,
			dropped:            true,
			logicalEffect:      EffectMove,
			logicalAvailable:   true,
			performedEffect:    EffectNone,
			performedAvailable: true,
			returnedEffect:     EffectCopy,
			want:               EffectMove,
		},
		{
			name:               "performed effect identifies unoptimized move",
			allowed:            EffectSetAll,
			dropped:            true,
			performedEffect:    EffectMove,
			performedAvailable: true,
			returnedEffect:     EffectCopy,
			want:               EffectMove,
		},
		{
			name:               "explicit none is not replaced by less reliable result",
			allowed:            EffectSetAll,
			dropped:            true,
			performedEffect:    EffectNone,
			performedAvailable: true,
			returnedEffect:     EffectMove,
			want:               EffectNone,
		},
		{
			name:           "OLE result is fallback when shell formats are absent",
			allowed:        EffectSetAll,
			dropped:        true,
			returnedEffect: EffectLink,
			want:           EffectLink,
		},
		{
			name:               "logical none remains unknown",
			allowed:            EffectSetAll,
			dropped:            true,
			logicalAvailable:   true,
			performedEffect:    EffectMove,
			performedAvailable: true,
			returnedEffect:     EffectMove,
			want:               EffectNone,
		},
		{
			name:             "cancelled drag has no effect",
			allowed:          EffectSetAll,
			logicalEffect:    EffectMove,
			logicalAvailable: true,
			returnedEffect:   EffectMove,
			want:             EffectNone,
		},
		{
			name:           "unallowed result stays unknown",
			allowed:        EffectSetCopy,
			dropped:        true,
			returnedEffect: EffectMove,
			want:           EffectNone,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resolveEffect(test.allowed, test.dropped, test.logicalEffect, test.logicalAvailable,
				test.performedEffect, test.performedAvailable, test.returnedEffect)
			if got != test.want {
				t.Fatalf("resolveEffect()=%d, want %d", got, test.want)
			}
		})
	}
}

func TestBeginRejectsEmptySelectionOnce(t *testing.T) {
	completions := 0
	Begin(nil, nil, func(result Result, err error) {
		completions++
		if result != "" || err == nil {
			t.Fatalf("result=%q err=%v", result, err)
		}
	})
	if completions != 1 {
		t.Fatalf("expected one completion, got %d", completions)
	}
}

func TestBeginWithOptionsRejectsEmptyAllowedEffectsOnce(t *testing.T) {
	completions := 0
	BeginWithOptions(nil, nil, Options{PreferredEffect: EffectCopy}, func(effect Effect, err error) {
		completions++
		if effect != EffectNone || !errors.Is(err, ErrInvalidEffectOptions) {
			t.Fatalf("effect=%d err=%v", effect, err)
		}
	})
	if completions != 1 {
		t.Fatalf("expected one completion, got %d", completions)
	}
}

func TestBeginWithOptionsRejectsUnsupportedEffectsBeforeNativeStart(t *testing.T) {
	if !Available || SupportedEffects()&EffectSetMove != 0 {
		t.Skip("requires a copy-only native backend")
	}
	for _, effect := range []Effect{EffectMove, EffectLink} {
		completions := 0
		BeginWithOptions(nil, nil, Options{
			AllowedEffects:  Effects(effect),
			PreferredEffect: effect,
		}, func(got Effect, err error) {
			completions++
			if got != EffectNone || !errors.Is(err, ErrUnsupportedEffect) {
				t.Fatalf("requested=%d got=%d err=%v", effect, got, err)
			}
		})
		if completions != 1 {
			t.Fatalf("effect=%d completions=%d", effect, completions)
		}
	}
}
