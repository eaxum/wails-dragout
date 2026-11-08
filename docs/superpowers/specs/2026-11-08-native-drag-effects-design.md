# Native Drag Effects Design

## Intent

Extend `wails-dragout` with a platform-neutral way to request and report Copy, Move, and Link operations, primarily so Windows drag-out can behave like Explorer and eventually fit a Wails-native implementation.

## Goals

- Keep the existing `Begin` API and its copy-only `Dropped`/`Cancelled` result behavior compatible.
- Add an opt-in API that accepts allowed effects and a preferred effect, and reports the resulting operation.
- On Windows, negotiate Copy, Move, and Link through OLE, use the preferred-effect clipboard format, and read completion formats that distinguish optimized from unoptimized moves.
- Never imply a confirmed operation when the platform does not provide a reliable result; an unknown effect must not authorize source deletion.
- Report per-platform effect capabilities. macOS and Linux remain copy-only in this change.

## Non-goals

- Deleting or moving the source files in this library.
- Implementing Move/Link on macOS or Linux.
- Changing existing `wails.Start` copy-only behavior or adding application-specific permission policy.

## Constraints

- Preserve Go 1.27.2 and Wails `v3.0.0-beta.32` compatibility.
- Keep platform-specific native code behind existing build tags and cgo backends.
- Keep old callers on copy-only behavior; opting into the new API is required to request other effects.
- A native drop acceptance without a confirmed effect is not a confirmed Move, Copy, or Link.

## Approaches

1. Add only Windows-specific flags and expose OLE results. Rejected because it makes the public API platform-specific and complicates eventual Wails integration.
2. Add a platform-neutral options/result API, preserve `Begin` as the copy-only compatibility path, and implement all three effects on Windows first. **Selected.** This matches the issue discussion, lets callers discover capabilities, and keeps other platforms honestly copy-only.
3. Change the existing `Begin` default to prefer Move everywhere. Rejected because it silently changes existing callers' behavior and conflicts with the compatibility direction in the issue discussion.

## Design

Export `EffectNone`, `EffectCopy`, `EffectMove`, and `EffectLink` with the Windows OLE bit values 0, 1, 2, and 4. `Effects` is a bit set with `EffectSetCopy`, `EffectSetMove`, `EffectSetLink`, and `EffectSetAll`. `Options` contains `AllowedEffects` and `PreferredEffect`. `SupportedEffects()` reports the current build's capabilities, and `BeginWithOptions(window, paths, options, complete func(Effect, error))` validates requests before dispatching to a backend.

Add `wails.StartWithOptions(ctx, window, paths, options) (Effect, error)` as the service-goroutine adapter for `BeginWithOptions`. It dispatches to the Wails UI thread and waits for completion while leaving existing `wails.Start` unchanged.

`BeginWithOptions` reports the selected operation through its effect callback. Internally, each backend also preserves the native `Dropped`/`Cancelled` result so legacy `Begin` can retain its existing status behavior independently of effect certainty. On Windows, legacy `Begin` remains `Dropped` only when `DoDragDrop` returns `DRAGDROP_S_DROP` with returned effect `COPY`.

Windows advertises the requested preference, offers the requested supported OLE effects, and reports the actual effect using `CFSTR_LOGICALPERFORMEDDROPEFFECT`, `CFSTR_PERFORMEDDROPEFFECT`, and a valid `DoDragDrop` result in that reliability order. An explicit `NONE` from a completion format does not fall through to a less reliable result. If the accepted drop has no reliable effect, the new API reports `EffectNone`; it does not delete anything. macOS and Linux accept Copy-only options and report Copy on successful drops. Unsupported requested effects return `ErrUnsupportedEffect`; unsupported builds preserve `ErrUnsupported`.

The Windows Shell data object receives `CFSTR_PREFERREDDROPEFFECT` through `IDataObject::SetData`, preserving the Shell file formats and target completion notifications. The library reports an effect only; it never deletes source paths. Callers must not use a legacy `Dropped` result or new `EffectNone` as confirmation of a Move.
