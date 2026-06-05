#define WASAPI_CAPTURE_EXPORTS
#include "wasapi_capture.h"

#include <windows.h>
#include <mmdeviceapi.h>
#include <audioclient.h>
#include <cmath>
#include <atomic>
#include <thread>
#include <string>
#include <sstream>
#include <mutex>
#include <chrono>

#pragma comment(lib, "ole32.lib")

namespace {

IMMDeviceEnumerator* g_enumerator = nullptr;
IMMDevice* g_device = nullptr;
IAudioClient* g_audioClient = nullptr;
IAudioCaptureClient* g_captureClient = nullptr;
WAVEFORMATEX* g_waveFormat = nullptr;

std::atomic<bool> g_running{false};
std::atomic<float> g_level{0.0f};
std::atomic<int> g_eventCount{0};
std::atomic<int> g_lastError{0};
std::thread g_captureThread;

// Ring buffer for captured audio.
// 1MB holds about 10s of 48kHz PCM16 mono audio, enough for one ASR segment.
const int RING_SIZE = 1024 * 1024;
unsigned char g_ringBuf[RING_SIZE];
int g_ringWrite = 0;   // write cursor
int g_ringTotal = 0;    // total bytes written (for diagnostic)
std::mutex g_ringMutex;

// Diagnostic string (updated on start, read by wasapi_get_diag)
std::string g_diag;
std::mutex g_diagMutex;

void updateDiag(int polls, int events, int dataPkts, int silentPkts) {
    std::lock_guard<std::mutex> lock(g_diagMutex);
    std::ostringstream oss;
    oss << "fmt:" << g_waveFormat->nSamplesPerSec << "Hz "
        << (int)g_waveFormat->nChannels << "ch "
        << (int)g_waveFormat->wBitsPerSample << "bit"
        << " | poll:" << polls
        << " evt:" << events
        << " dat:" << dataPkts
        << " sil:" << silentPkts;
    g_diag = oss.str();
}

void captureLoop() {
    // Use polling mode — more reliable than event-driven for loopback
    HRESULT hr = g_audioClient->Start();
    if (FAILED(hr)) {
        g_lastError.store(hr);
        return;
    }

    int pollCount = 0;
    int eventCount = 0;
    int dataPacketCount = 0;
    int silentPacketCount = 0;
    auto lastDiagUpdate = std::chrono::steady_clock::now();

    while (g_running.load()) {
        pollCount++;
        Sleep(10); // ~100 polls/sec

        UINT32 packetLength = 0;
        hr = g_captureClient->GetNextPacketSize(&packetLength);
        if (FAILED(hr)) {
            g_lastError.store(hr);
            break;
        }

        float maxSample = 0.0f;

        while (packetLength > 0) {
            BYTE* data;
            UINT32 numFrames;
            DWORD flags;
            hr = g_captureClient->GetBuffer(&data, &numFrames, &flags, nullptr, nullptr);
            if (FAILED(hr)) {
                g_lastError.store(hr);
                goto cleanup;
            }

            eventCount++;

            if (flags & AUDCLNT_BUFFERFLAGS_SILENT) {
                silentPacketCount++;
            } else {
                dataPacketCount++;
                if (data) {
                    float* samples = reinterpret_cast<float*>(data);
                    UINT32 sampleCount = numFrames * g_waveFormat->nChannels;
                    for (UINT32 i = 0; i < sampleCount; i++) {
                        float absVal = std::fabs(samples[i]);
                        if (absVal > maxSample) maxSample = absVal;
                    }

                    // Ring buffer: float32→PCM16, multichannel→mono
                    {
                        std::lock_guard<std::mutex> lock(g_ringMutex);
                        for (UINT32 i = 0; i < numFrames; i++) {
                            float mono = 0.0f;
                            for (UINT32 ch = 0; ch < g_waveFormat->nChannels; ch++) {
                                mono += samples[i * g_waveFormat->nChannels + ch];
                            }
                            mono /= g_waveFormat->nChannels;
                            int s16 = (int)(mono * 32767.0f);
                            if (s16 > 32767) s16 = 32767;
                            if (s16 < -32768) s16 = -32768;
                            short s = (short)s16;
                            g_ringBuf[g_ringWrite] = (unsigned char)(s & 0xFF);
                            g_ringWrite = (g_ringWrite + 1) % RING_SIZE;
                            g_ringBuf[g_ringWrite] = (unsigned char)((s >> 8) & 0xFF);
                            g_ringWrite = (g_ringWrite + 1) % RING_SIZE;
                            g_ringTotal += 2;
                        }
                    }
                }
            }

            hr = g_captureClient->ReleaseBuffer(numFrames);
            if (FAILED(hr)) {
                g_lastError.store(hr);
                goto cleanup;
            }

            hr = g_captureClient->GetNextPacketSize(&packetLength);
            if (FAILED(hr)) break;
        }

        g_level.store(maxSample, std::memory_order_relaxed);

        // Update diag every ~500ms
        auto now = std::chrono::steady_clock::now();
        if (std::chrono::duration_cast<std::chrono::milliseconds>(now - lastDiagUpdate).count() > 500) {
            updateDiag(pollCount, eventCount, dataPacketCount, silentPacketCount);
            lastDiagUpdate = now;
        }
    }

cleanup:
    updateDiag(pollCount, eventCount, dataPacketCount, silentPacketCount);
    g_audioClient->Stop();
}

void cleanup() {
    if (g_captureClient) { g_captureClient->Release(); g_captureClient = nullptr; }
    if (g_audioClient) { g_audioClient->Release(); g_audioClient = nullptr; }
    if (g_device) { g_device->Release(); g_device = nullptr; }
    if (g_enumerator) { g_enumerator->Release(); g_enumerator = nullptr; }
    if (g_waveFormat) { CoTaskMemFree(g_waveFormat); g_waveFormat = nullptr; }
}

} // anonymous namespace

WASAPI_API int wasapi_start() {
    if (g_running.load()) return -1;

    HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
    bool comInitialized = SUCCEEDED(hr);

    hr = CoCreateInstance(
        __uuidof(MMDeviceEnumerator), nullptr, CLSCTX_ALL,
        __uuidof(IMMDeviceEnumerator), (void**)&g_enumerator);
    if (FAILED(hr)) {
        if (comInitialized) CoUninitialize();
        return 1;
    }

    hr = g_enumerator->GetDefaultAudioEndpoint(eRender, eConsole, &g_device);
    if (FAILED(hr)) { cleanup(); if (comInitialized) CoUninitialize(); return 2; }

    hr = g_device->Activate(__uuidof(IAudioClient), CLSCTX_ALL,
        nullptr, (void**)&g_audioClient);
    if (FAILED(hr)) { cleanup(); if (comInitialized) CoUninitialize(); return 3; }

    hr = g_audioClient->GetMixFormat(&g_waveFormat);
    if (FAILED(hr)) { cleanup(); if (comInitialized) CoUninitialize(); return 4; }

    // Build format diagnostic string
    {
        std::lock_guard<std::mutex> lock(g_diagMutex);
        std::ostringstream oss;
        oss << "fmt:" << g_waveFormat->nSamplesPerSec << "Hz "
            << (int)g_waveFormat->nChannels << "ch "
            << (int)g_waveFormat->wBitsPerSample << "bit "
            << "tag:0x" << std::hex << g_waveFormat->wFormatTag;
        g_diag = oss.str();
    }

    hr = g_audioClient->Initialize(
        AUDCLNT_SHAREMODE_SHARED,
        AUDCLNT_STREAMFLAGS_LOOPBACK,
        0, 0, g_waveFormat, nullptr);
    if (FAILED(hr)) { cleanup(); if (comInitialized) CoUninitialize(); return 5; }

    hr = g_audioClient->GetService(
        __uuidof(IAudioCaptureClient), (void**)&g_captureClient);
    if (FAILED(hr)) { cleanup(); if (comInitialized) CoUninitialize(); return 6; }

    g_running.store(true);
    g_level.store(0.0f);
    g_eventCount.store(0);
    g_lastError.store(0);
    g_captureThread = std::thread(captureLoop);

    return 0;
}

WASAPI_API void wasapi_stop() {
    g_running.store(false);
    if (g_captureThread.joinable()) {
        g_captureThread.join();
    }
    cleanup();
    g_level.store(-1.0f);
}

WASAPI_API float wasapi_get_level() {
    return g_level.load(std::memory_order_relaxed);
}

WASAPI_API int wasapi_is_capturing() {
    return g_running.load() ? 1 : 0;
}

WASAPI_API const char* wasapi_get_diag() {
    std::lock_guard<std::mutex> lock(g_diagMutex);
    // Return pointer to static copy (safe because caller uses immediately)
    static thread_local std::string copy;
    copy = g_diag;
    return copy.c_str();
}

WASAPI_API int wasapi_get_last_error() {
    return g_lastError.load();
}

// Pre-allocated output buffer (avoids caller-side allocation)
static unsigned char g_readBuf[RING_SIZE];
static int g_readBytes = 0;

WASAPI_API int wasapi_read_audio() {
    std::lock_guard<std::mutex> lock(g_ringMutex);
    int available = (g_ringTotal < RING_SIZE) ? g_ringTotal : RING_SIZE;
    int toCopy = (available < (int)sizeof(g_readBuf)) ? available : (int)sizeof(g_readBuf);

    int readPos = (g_ringWrite - toCopy + RING_SIZE) % RING_SIZE;
    for (int i = 0; i < toCopy; i++) {
        g_readBuf[i] = g_ringBuf[(readPos + i) % RING_SIZE];
    }

    g_ringTotal = 0;
    g_ringWrite = 0;
    g_readBytes = toCopy;
    return toCopy;
}

WASAPI_API const unsigned char* wasapi_get_audio_data() {
    return g_readBuf;
}

WASAPI_API int wasapi_available_bytes() {
    std::lock_guard<std::mutex> lock(g_ringMutex);
    if (g_ringTotal < RING_SIZE) return g_ringTotal;
    return RING_SIZE;
}
