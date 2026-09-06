# wails-dragout

Copy-only native file dragging for Wails v3, maintained by Eaxum.

The root `dragout` package implements Windows OLE, macOS AppKit, and Linux GTK3 backends. The `wails` subpackage starts native work on the Wails UI thread and waits for completion on the calling service goroutine. No frontend component, asset database, authentication, or application-specific permission model is included.

## Status

Windows and macOS were manually verified in Clustta before extraction. Linux is experimental: native compilation and X11/Wayland desktop testing remain pending. Automated tests do not simulate user-driven OS drops. The public module is published as a preview. Linux remains unverified.

The module currently targets Go 1.25.8 and Wails `v3.0.0-alpha.56`, matching the host application at extraction. Wails alpha compatibility is intentionally pinned; newer versions need verification.

## Wails usage

From a Wails service method, obtain the trusted calling window from the binding context and resolve/authorize your own files:

```go
import (
    "context"
    "errors"

    "github.com/eaxum/wails-dragout"
    wailsdrag "github.com/eaxum/wails-dragout/wails"
    "github.com/wailsapp/wails/v3/pkg/application"
)

func exportFiles(ctx context.Context, validatedPaths []string) (dragout.Result, error) {
    window, ok := ctx.Value(application.WindowKey).(*application.WebviewWindow)
    if !ok || window == nil {
        return "", errors.New("dragging requires a desktop window")
    }
    return wailsdrag.Start(ctx, window, validatedPaths)
}
```

Call the adapter from a service goroutine, never the UI thread. Initiate while the left mouse button is still held, typically from a frontend `dragstart` handler that calls `preventDefault()`. Serialize export requests in the host until completion. Clustta's `DragOutService` already does this.

The host must verify access, asset/path associations, and project path containment before calling this package. The native backends perform basic input checks but do not impose application permissions or project boundaries.

## API and completion

- `dragout.Available` reports compiled platform support. It does not certify desktop/compositor compatibility.
- `dragout.Begin(window, paths, complete)` starts on the platform UI thread. `window` is a Windows HWND, macOS NSWindow, or Linux GtkWindow pointer. The non-nil completion callback must return promptly. Windows may block until OLE ends; macOS and Linux complete asynchronously. Completion may also occur immediately on failure.
- `wailsdrag.Start(ctx, window, paths)` handles main-thread dispatch and returns a `dragout.Result` or error.
- `dragout.Dropped` means the native drag was accepted. It does not guarantee the target has completed importing/copying.
- `dragout.Cancelled` covers cancellation or rejection. Unsupported builds return `dragout.ErrUnsupported`.

Context cancellation is checked before initiation. Once started, the adapter waits for native completion even if the context is cancelled, preserving resource lifetime and the host's session guard. Escape cancellation is handled by the OS. There is no separate programmatic cancellation API.

## Build and test

All supported native backends require cgo and the usual Wails platform toolchain:

| Platform | Native toolchain | Verification |
| --- | --- | --- |
| Windows | C compiler and Windows system libraries | OLE payload tests and prior Clustta manual checks |
| macOS | Xcode command-line tools and Cocoa | Prior Clustta manual checks |
| Linux | C compiler, pkg-config, GTK3 development files; WebKitGTK 4.1 for the Wails adapter | Pending |

```sh
go test ./...
```

To test only the native package, use `go test .`. Tests include Unicode paths, invalid input, completion behavior, and platform payload checks. Builds without cgo and unsupported platforms expose the unsupported fallback.

Repeat the manual checks in the host after extraction: multiple files and directories, Unicode/special filenames, file-manager and DCC drops, Escape, rejected drops, quick release, repeated sessions, modifier keys, and source-window closure. Sources must stay intact. Linux requires separate X11 and native Wayland checks; XWayland alone is insufficient. GTK currently initiates without the original mouse event because the webview binding is asynchronous, so Wayland input-serial behavior requires testing.

## Local development and release

For local package development, use a sibling checkout and a temporary host-module replacement:

```go
require github.com/eaxum/wails-dragout v0.0.0
replace github.com/eaxum/wails-dragout => ../wails-dragout
```

`v0.0.0` is only a local placeholder. Do not distribute a host build configuration that depends on this sibling path. After publishing a verified tag, remove the replacement, require that tag, and run `go mod tidy` and the host checks. Never present Linux support as verified until its checks pass.

## License and attribution

Eaxum contributions retain the AGPL-3.0 license copied from Clustta; see `LICENSE` and `NOTICE`. The adapted drag-rs portions retain their original MIT notice in `LICENSE.drag-rs`. Distribute both notices with the adapted code.

Native platform implementations draw on [drag-rs](https://github.com/spacedriveapp/drag-rs/tree/35739c3e5e5b05165ce08947d7148357e8dbe640), with application-specific integration originally developed in Clustta.

## Default drag images

Windows attaches a 32-pixel system file icon using the registered file-type icon and shell drag-image helper. It does not open the asset or extract a thumbnail, and falls back to the standard cursor if the icon is unavailable. Multiple-file selections use the first file icon. macOS uses its existing system file icons; Linux uses the GTK default. No frontend image or custom preview API is required.
