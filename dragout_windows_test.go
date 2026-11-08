//go:build windows && cgo

package dragout

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestShellDataObject(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	root := t.TempDir()
	var paths []string
	for _, name := range []string{"first", "second"} {
		directory := filepath.Join(root, name)
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, "test space 日本語.blend")
		if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	for _, files := range [][]string{paths[:1], paths} {
		if err := probeFiles(files); err != nil {
			t.Fatal(err)
		}
	}
	if err := probeFiles([]string{filepath.Join(root, "missing.blend")}); err == nil {
		t.Fatal("missing file accepted")
	}
}

func TestPreferredDropEffectFormatRoundTrip(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	path := filepath.Join(t.TempDir(), "preferred-effect.txt")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []Effect{EffectCopy, EffectMove, EffectLink} {
		if err := probePreferredEffect([]string{path}, effect); err != nil {
			t.Fatalf("effect %d: %v", effect, err)
		}
	}
}

func TestPerformedDropEffectFormatsRoundTrip(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	path := filepath.Join(t.TempDir(), "performed-effect.txt")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name               string
		logicalEffect      Effect
		logicalAvailable   bool
		performedEffect    Effect
		performedAvailable bool
	}{
		{
			name:               "optimized move",
			logicalEffect:      EffectMove,
			logicalAvailable:   true,
			performedEffect:    EffectNone,
			performedAvailable: true,
		},
		{
			name:               "unoptimized move",
			logicalAvailable:   false,
			performedEffect:    EffectMove,
			performedAvailable: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := probeDropEffectFormats([]string{path}, test.logicalEffect, test.logicalAvailable,
				test.performedEffect, test.performedAvailable); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEncodePathsRejectsInvalidInput(t *testing.T) {
	for _, paths := range [][]string{nil, {"relative.blend"}, {"C:\\bad\x00.blend"}} {
		if _, err := encodePaths(paths); err == nil {
			t.Fatalf("accepted %q", paths)
		}
	}
}
