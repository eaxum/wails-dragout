#pragma once
#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif
int32_t clustta_drag_files(uintptr_t window, const uint16_t *paths, size_t count,
    uint32_t allowedEffects, uint32_t preferredEffect, uint32_t *returnedEffect,
    uint32_t *logicalEffect, int32_t *hasLogicalEffect,
    uint32_t *performedEffect, int32_t *hasPerformedEffect);
int32_t clustta_probe_drag_files(const uint16_t *paths, size_t count);
int32_t clustta_probe_preferred_effect(const uint16_t *paths, size_t count, uint32_t effect);
int32_t clustta_probe_drop_effect_formats(const uint16_t *paths, size_t count,
    uint32_t logicalEffect, int32_t hasLogicalEffect,
    uint32_t performedEffect, int32_t hasPerformedEffect);
#ifdef __cplusplus
}
#endif
