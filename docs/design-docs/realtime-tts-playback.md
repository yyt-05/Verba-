# Realtime TTS Playback

## Goal

Realtime Chinese speech should sound continuous and coherent. It may be slightly behind the latest subtitle, but it must not skip older queued text just to catch up with newer translation output.

## Playback Policy

- Preserve text order in the TTS queue.
- Do not clear queued text or reset audio merely because newer text arrives.
- Preserve queued PCM audio on the client; never trim from the front of unplayed audio to catch up.
- Prefer slightly larger Chinese chunks over very small low-latency fragments.
- Short completed phrases should wait for more context unless the session is flushed.
- When a long phrase has no punctuation, use a timeout fallback so speech can still progress.

## Current Thresholds

- Minimum chunk: 18 runes.
- Target chunk: 44 runes.
- Soft maximum: 64 runes.
- Hard maximum: 88 runes.
- First chunk wait: 1200 ms.
- Normal chunk wait: 1500 ms.
- Client startup prebuffer: 800 ms.
- Client target device queue: 1000 ms.
- Client waveOut block size: 80 ms.

These values intentionally trade a small amount of latency for smoother speech.
