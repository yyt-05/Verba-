#pragma once

#ifdef WASAPI_CAPTURE_EXPORTS
#define WASAPI_API __declspec(dllexport)
#else
#define WASAPI_API __declspec(dllimport)
#endif

extern "C" {

/// Start capturing system audio via WASAPI loopback.
/// Returns 0 on success, non-zero on failure.
WASAPI_API int wasapi_start();

/// Stop capturing and release resources.
WASAPI_API void wasapi_stop();

/// Get the current audio level (RMS amplitude), range 0.0 - 1.0.
/// Returns -1.0 if capture is not running.
WASAPI_API float wasapi_get_level();

/// Check if capture is currently active.
/// Returns 1 if capturing, 0 if not.
WASAPI_API int wasapi_is_capturing();

/// Get diagnostic info: audio format, buffer size, last error.
/// Caller must NOT free the returned string.
WASAPI_API const char* wasapi_get_diag();

/// Get the last error code from the capture loop (0 = no error).
WASAPI_API int wasapi_get_last_error();

/// Drain the ring buffer into internal output buffer. Returns number of bytes copied.
WASAPI_API int wasapi_read_audio();

/// Get the pointer to the output buffer filled by the last wasapi_read_audio call.
WASAPI_API const unsigned char* wasapi_get_audio_data();

/// Get number of bytes currently available in the capture buffer.
WASAPI_API int wasapi_available_bytes();

}
