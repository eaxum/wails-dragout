# Native Drag Effects Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a backward-compatible platform-neutral drag-effects API and Windows Copy/Move/Link support with safe outcome reporting.

**Architecture:** Add shared effect/options types and a new callback API while retaining `Begin` as a copy-only wrapper. Windows passes the allowed effects and preference through Go to OLE and resolves completion from the Shell data object; macOS/Linux remain copy-only and reject unsupported requests.

**Tech Stack:** Go 1.27.2, cgo, Windows OLE/Shell IDataObject, existing Darwin AppKit and Linux GTK backends.

**Spec:** `docs/superpowers/specs/2026-11-08-native-drag-effects-design.md`

## Global Constraints

- Preserve Go 1.27.2 and Wails `v3.0.0-beta.32` compatibility.
- Keep existing `Begin` callers on copy-only behavior and retain `Dropped`/`Cancelled` results.
- macOS and Linux expose Copy only in this change.
- An accepted drop alone never reports a confirmed operation or authorizes source deletion.

## Review Focus

- An optimized Windows Move reports Move from `CFSTR_LOGICALPERFORMEDDROPEFFECT` even when `CFSTR_PERFORMEDDROPEFFECT` is NONE.
- An unoptimized Windows Move reports Move from `CFSTR_PERFORMEDDROPEFFECT`.
- Cancellation, rejection, or an ambiguous accepted drop never reports a confirmed Move.
- Invalid or unsupported effect combinations return an error before native drag initiation.
- Existing `Begin` behavior remains Copy-only and keeps its legacy completion values.

---

### Task 1: Define and validate the portable API

**Files:**
- Modify: `dragout.go`
- Modify: `begin_test.go`

**Interfaces:**
- Produces `Effect`, `Effects`, `Options`, `SupportedEffects() Effects`, and `validateOptions(options Options, supported Effects) error`.
- `EffectNone` represents cancellation or an accepted drop whose operation cannot be confirmed; validation errors use a non-nil error.

- [x] Add tests for allowed/preferred effect validation and capability exposure.
- [x] Run the focused Go tests and confirm the new API tests fail before implementation.
- [x] Implement common effect types, capability reporting, and validation.
- [x] Run the focused Go tests; confirm validation and existing `Begin` behavior pass.

### Task 2: Implement Windows OLE effect negotiation and outcome reporting

**Files:**
- Modify: `dragout_windows.go`
- Modify: `dragout_windows.c`
- Modify: `dragout_windows.h`
- Modify: `dragout_windows_test.go`

**Interfaces:**
- Consumes Task 1's `Options`, `Effect`, and validation contract.
- Windows reports Copy, Move, Link, or `EffectNone`; it never removes source paths.
- Produces the Windows `startWithOptions` native bridge consumed by Task 3.

- [x] Add focused Windows checks for preferred-effect advertisement and outcome selection, including native completion-format round trips for logical/optimized and performed/unoptimized moves.
- [x] Run the focused Windows checks; the first cgo compile exposed `DWORD*`/`uint32_t*` incompatibility, fixed by using `uint32_t` for 32-bit clipboard payloads.
- [x] Pass allowed effects to `DoDragDrop`; advertise the preferred Shell effect while preserving Shell file formats and its completion `SetData` notifications.
- [x] Resolve the returned operation in the order specified by the design and return unknown when no reliable effect exists.
- [x] Run the focused Windows checks with Windows cgo enabled; all preferred/completion format round trips passed.

### Task 3: Keep non-Windows backends explicitly Copy-only

**Files:**
- Modify: `dragout.go`
- Modify: `dragout_darwin.go`
- Modify: `dragout_linux.go`
- Modify: `dragout_unsupported.go`
- Modify: `begin_sync.go`
- Modify: `dragout_darwin_test.go`
- Modify: `dragout_linux_test.go`

**Interfaces:**
- Consumes Task 1's API contract; each supported non-Windows backend reports Copy only.
- Unsupported builds report no supported effects and preserve `ErrUnsupported` behavior.
- Produces the common `BeginWithOptions` dispatcher and the legacy Copy-only `Begin` wrapper over each backend.

- [x] Add checks that Move/Link requests are rejected on copy-only platforms and accepted Copy requests report Copy.
- [x] Run focused platform checks and confirm unsupported-effect checks fail before implementation (new API test first failed because `BeginWithOptions` was not yet defined).
- [x] Implement the copy-only adapters and unsupported-build behavior.
- [x] Run `go test ./... -count=1` on Windows/amd64 with cgo enabled; the Windows backend passed. Darwin/Linux cgo builds remain platform-specific.

### Task 4: Expose effect options through the Wails adapter

**Files:**
- Modify: `wails/adapter.go`
- Create: `wails/adapter_test.go`

**Interfaces:**
- Consumes Task 1's `Options` and `Effect` and Task 3's `BeginWithOptions`.
- Produces `StartWithOptions(ctx context.Context, window *application.WebviewWindow, paths []string, options dragout.Options) (dragout.Effect, error)` without changing legacy `Start`.

- [x] Add a test proving a cancelled context returns before native initiation.
- [x] Run the focused adapter test and confirm it fails before the adapter exists.
- [x] Implement UI-thread dispatch and completion waiting in `wails/adapter.go`.
- [x] Run the focused adapter test and the full Wails-dragout tests (30 passed in 2 packages with Windows cgo enabled).

### Task 5: Document capabilities, compatibility, and safe completion handling

**Files:**
- Modify: `README.md`

- [x] Document the new options/capability API, Windows effects, copy-only compatibility path, and the rule that unknown outcomes must not trigger source deletion.
- [x] Run `go test ./... -count=1` (30 passed in 2 packages with Windows cgo enabled and 15 popup tests passed); build the Windows popup helper from the OrganizerV2 worktree.
- [x] Review the final diff against the design and check for whitespace errors.
