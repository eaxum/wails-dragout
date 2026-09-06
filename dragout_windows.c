//go:build windows && cgo

// Copyright (c) 2026 Eaxum.
// Adapted from drag-rs; see NOTICE and LICENSE.drag-rs.
#define COBJMACROS
#include "dragout_windows.h"
#include <windows.h>
#include <shlobj.h>
#include <stdlib.h>

typedef struct {
    IDropSource base;
    LONG references;
    HWND window;
} FileDropSource;

static ULONG STDMETHODCALLTYPE sourceAddRef(IDropSource *self) {
    return InterlockedIncrement(&((FileDropSource *)self)->references);
}

static ULONG STDMETHODCALLTYPE sourceRelease(IDropSource *self) {
    ULONG remaining = InterlockedDecrement(&((FileDropSource *)self)->references);
    if (!remaining) free(self);
    return remaining;
}

static HRESULT STDMETHODCALLTYPE sourceQueryInterface(IDropSource *self, REFIID iid, void **object) {
    if (!object) return E_POINTER;
    *object = NULL;
    if (!IsEqualIID(iid, &IID_IUnknown) && !IsEqualIID(iid, &IID_IDropSource)) return E_NOINTERFACE;
    *object = self;
    sourceAddRef(self);
    return S_OK;
}

static HRESULT STDMETHODCALLTYPE sourceContinue(IDropSource *self, BOOL escape, DWORD keys) {
    if (escape || !IsWindow(((FileDropSource *)self)->window)) return DRAGDROP_S_CANCEL;
    return keys & MK_LBUTTON ? S_OK : DRAGDROP_S_DROP;
}

static HRESULT STDMETHODCALLTYPE sourceFeedback(IDropSource *self, DWORD effect) {
    return DRAGDROP_S_USEDEFAULTCURSORS;
}

static IDropSourceVtbl sourceMethods = {
    sourceQueryInterface, sourceAddRef, sourceRelease, sourceContinue, sourceFeedback
};

static HRESULT fileDataObject(const uint16_t *paths, size_t count, IDataObject **data) {
    if (!paths || !count || count > UINT_MAX) return E_INVALIDARG;
    PIDLIST_ABSOLUTE *items = calloc(count, sizeof(*items));
    if (!items) return E_OUTOFMEMORY;
    const wchar_t *path = (const wchar_t *)paths;
    HRESULT result = S_OK;
    for (size_t index = 0; index < count; ++index) {
        result = SHParseDisplayName(path, NULL, &items[index], 0, NULL);
        if (FAILED(result)) break;
        path += wcslen(path) + 1;
    }
    IShellItemArray *array = NULL;
    if (SUCCEEDED(result)) {
        result = SHCreateShellItemArrayFromIDLists((UINT)count, (PCIDLIST_ABSOLUTE *)items, &array);
    }
    if (SUCCEEDED(result)) {
        result = IShellItemArray_BindToHandler(array, NULL, &BHID_DataObject, &IID_IDataObject, (void **)data);
        IShellItemArray_Release(array);
    }
    for (size_t index = 0; index < count; ++index) CoTaskMemFree(items[index]);
    free(items);
    return result;
}

static const LONG DragIconSize = 32;
static const UINT DragBitmapBits = 32;
static const UINT FullAlpha = 255;
static const DWORD OpaqueBlack = 0xff000000;
static const DWORD OpaqueWhite = 0xffffffff;

static HBITMAP renderIcon(HICON icon, DWORD background, RGBQUAD **pixels) {
    BITMAPINFO info = {0};
    info.bmiHeader.biSize = sizeof(info.bmiHeader);
    info.bmiHeader.biWidth = DragIconSize;
    info.bmiHeader.biHeight = -DragIconSize;
    info.bmiHeader.biPlanes = 1;
    info.bmiHeader.biBitCount = DragBitmapBits;
    info.bmiHeader.biCompression = BI_RGB;
    HDC device = CreateCompatibleDC(NULL);
    if (!device) return NULL;
    HBITMAP image = CreateDIBSection(device, &info, DIB_RGB_COLORS, (void **)pixels, NULL, 0);
    if (!image) {
        DeleteDC(device);
        return NULL;
    }
    for (LONG index = 0; index < DragIconSize * DragIconSize; ++index) {
        ((DWORD *)*pixels)[index] = background;
    }
    HGDIOBJ previous = SelectObject(device, image);
    BOOL rendered = DrawIconEx(device, 0, 0, icon, DragIconSize, DragIconSize, 0, NULL, DI_NORMAL);
    SelectObject(device, previous);
    GdiFlush();
    DeleteDC(device);
    if (!rendered) {
        DeleteObject(image);
        return NULL;
    }
    return image;
}

static HBITMAP iconBitmap(HICON icon) {
    RGBQUAD *blackPixels = NULL, *whitePixels = NULL;
    HBITMAP black = renderIcon(icon, OpaqueBlack, &blackPixels);
    if (!black) return NULL;
    HBITMAP white = renderIcon(icon, OpaqueWhite, &whitePixels);
    if (!white) {
        DeleteObject(black);
        return NULL;
    }
    // Two backgrounds recover alpha for both legacy masked and modern alpha icons.
    for (LONG index = 0; index < DragIconSize * DragIconSize; ++index) {
        RGBQUAD *pixel = &blackPixels[index];
        int difference = (int)whitePixels[index].rgbBlue - pixel->rgbBlue;
        UINT alpha = FullAlpha - max(0, difference);
        pixel->rgbReserved = (BYTE)alpha;
        if (!alpha) {
            pixel->rgbRed = pixel->rgbGreen = pixel->rgbBlue = 0;
            continue;
        }
        // InitializeFromBitmap expects straight alpha and premultiplies it itself.
        pixel->rgbRed = (BYTE)min(FullAlpha, pixel->rgbRed * FullAlpha / alpha);
        pixel->rgbGreen = (BYTE)min(FullAlpha, pixel->rgbGreen * FullAlpha / alpha);
        pixel->rgbBlue = (BYTE)min(FullAlpha, pixel->rgbBlue * FullAlpha / alpha);
    }
    DeleteObject(white);
    return black;
}

static HRESULT attachFileIcon(const uint16_t *paths, IDataObject *data) {
    SHFILEINFOW fileInfo = {0};
    // Resolve the registered file-type icon without opening the asset or extracting a thumbnail.
    if (!SHGetFileInfoW((const wchar_t *)paths, FILE_ATTRIBUTE_NORMAL, &fileInfo, sizeof(fileInfo),
        SHGFI_ICON | SHGFI_LARGEICON | SHGFI_USEFILEATTRIBUTES)) return E_FAIL;
    SHDRAGIMAGE image = {0};
    image.hbmpDragImage = iconBitmap(fileInfo.hIcon);
    DestroyIcon(fileInfo.hIcon);
    if (!image.hbmpDragImage) return E_FAIL;
    image.sizeDragImage.cx = image.sizeDragImage.cy = DragIconSize;
    image.ptOffset.x = image.ptOffset.y = DragIconSize / 2;
    image.crColorKey = CLR_NONE;
    IDragSourceHelper *helper = NULL;
    HRESULT result = CoCreateInstance(&CLSID_DragDropHelper, NULL, CLSCTX_INPROC_SERVER,
        &IID_IDragSourceHelper, (void **)&helper);
    if (FAILED(result)) {
        DeleteObject(image.hbmpDragImage);
        return result;
    }
    // InitializeFromBitmap takes ownership of the supplied bitmap.
    result = IDragSourceHelper_InitializeFromBitmap(helper, &image, data);
    IDragSourceHelper_Release(helper);
    return result;
}

int32_t clustta_drag_files(uintptr_t handle, const uint16_t *paths, size_t count) {
    HWND window = (HWND)handle;
    if (!IsWindow(window) || GetForegroundWindow() != window || GetAsyncKeyState(VK_LBUTTON) >= 0) {
        return DRAGDROP_S_CANCEL;
    }
    HRESULT result = OleInitialize(NULL);
    if (FAILED(result)) return result;
    IDataObject *data = NULL;
    result = fileDataObject(paths, count, &data);
    if (SUCCEEDED(result)) {
        FileDropSource *source = calloc(1, sizeof(*source));
        if (!source) {
            result = E_OUTOFMEMORY;
        } else {
            source->base.lpVtbl = &sourceMethods;
            source->references = 1;
            source->window = window;
            DWORD effect = DROPEFFECT_NONE;
            if (FAILED(attachFileIcon(paths, data))) {
                OutputDebugStringW(L"wails-dragout: system icon unavailable; using the default drag cursor.\n");
            }
            if (GetAsyncKeyState(VK_LBUTTON) >= 0) {
                result = DRAGDROP_S_CANCEL;
            } else {
                result = DoDragDrop(data, &source->base, DROPEFFECT_COPY, &effect);
                if (result == DRAGDROP_S_DROP && effect != DROPEFFECT_COPY) result = DRAGDROP_S_CANCEL;
            }
            sourceRelease(&source->base);
        }
        IDataObject_Release(data);
    }
    OleUninitialize();
    return result;
}

int32_t clustta_probe_drag_files(const uint16_t *paths, size_t count) {
    HRESULT result = OleInitialize(NULL);
    if (FAILED(result)) return result;
    IDataObject *data = NULL;
    result = fileDataObject(paths, count, &data);
    if (SUCCEEDED(result)) {
        result = attachFileIcon(paths, data);
        if (FAILED(result)) {
            IDataObject_Release(data);
            OleUninitialize();
            return result;
        }
        FORMATETC imageFormat = {RegisterClipboardFormatW(L"DragImageBits"), NULL, DVASPECT_CONTENT, -1, TYMED_HGLOBAL};
        result = IDataObject_QueryGetData(data, &imageFormat);
        if (FAILED(result)) {
            IDataObject_Release(data);
            OleUninitialize();
            return result;
        }
        FORMATETC format = {CF_HDROP, NULL, DVASPECT_CONTENT, -1, TYMED_HGLOBAL};
        STGMEDIUM medium = {0};
        result = IDataObject_GetData(data, &format, &medium);
        if (SUCCEEDED(result)) {
            if (DragQueryFileW((HDROP)medium.hGlobal, UINT_MAX, NULL, 0) != count) result = E_FAIL;
            ReleaseStgMedium(&medium);
        }
        IDataObject_Release(data);
    }
    OleUninitialize();
    return result;
}
