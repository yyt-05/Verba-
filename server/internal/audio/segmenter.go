package audio

import (
	"encoding/binary"
	"math"
	"sync"
)

const (
	defaultVoiceThreshold = 500
	defaultSilenceMs      = 700
	defaultMinVoiceMs     = 800
	defaultMaxSegmentMs   = 10000
)

// Segmenter buffers PCM16 mono chunks and returns a segment when speech ends.
type Segmenter struct {
	sampleRate     int
	voiceThreshold int
	silenceMsLimit int
	minVoiceMs     int
	maxSegmentMs   int

	buffer   []byte
	speaking bool
	silentMs int
	voicedMs int

	mu sync.Mutex
}

func NewSegmenter(sampleRate int) *Segmenter {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	return &Segmenter{
		sampleRate:     sampleRate,
		voiceThreshold: defaultVoiceThreshold,
		silenceMsLimit: defaultSilenceMs,
		minVoiceMs:     defaultMinVoiceMs,
		maxSegmentMs:   defaultMaxSegmentMs,
		buffer:         make([]byte, 0, sampleRate*2*4),
	}
}

func (s *Segmenter) AddPCM(chunk []byte) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(chunk) < 2 {
		return nil, false
	}

	durationMs := len(chunk) * 1000 / (s.sampleRate * 2)
	if durationMs <= 0 {
		durationMs = 1
	}

	voiced := pcmRMS(chunk) >= float64(s.voiceThreshold)
	if voiced {
		s.speaking = true
		s.silentMs = 0
		s.voicedMs += durationMs
		s.buffer = append(s.buffer, chunk...)
	} else if s.speaking {
		s.silentMs += durationMs
		s.buffer = append(s.buffer, chunk...)
	} else {
		return nil, false
	}

	if s.speaking && s.voicedMs >= s.maxSegmentMs {
		return s.flushLocked()
	}

	if s.speaking && s.silentMs >= s.silenceMsLimit {
		if s.voicedMs < s.minVoiceMs {
			s.resetLocked()
			return nil, false
		}
		return s.flushLocked()
	}

	return nil, false
}

func (s *Segmenter) flushLocked() ([]byte, bool) {
	segment := make([]byte, len(s.buffer))
	copy(segment, s.buffer)
	s.resetLocked()
	return segment, true
}

func (s *Segmenter) resetLocked() {
	s.buffer = s.buffer[:0]
	s.speaking = false
	s.silentMs = 0
	s.voicedMs = 0
}

func pcmRMS(pcm []byte) float64 {
	samples := len(pcm) / 2
	if samples == 0 {
		return 0
	}

	var sum float64
	for i := 0; i+1 < len(pcm); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
		v := float64(sample)
		sum += v * v
	}
	return math.Sqrt(sum / float64(samples))
}
