package audio

import (
	"io"
	"log"
)

// Chunk represents an audio chunk uploaded by the client.
type Chunk struct {
	Seq       int
	Data      []byte
	SessionID string
}

// Validate checks basic constraints on an audio chunk.
func (c *Chunk) Validate() error {
	if len(c.Data) == 0 {
		return nil // silent chunk, skip without error
	}
	if len(c.Data) > 2*1024*1024 {
		return io.ErrShortBuffer // reject chunks > 2MB
	}
	return nil
}

// Processor holds the pipeline processing logic.
// Phase 0: validates and accumulates chunks.
// Phase 1: triggers ASR when enough audio is buffered.
type Processor struct {
	buffer []byte
	seqLow int // oldest buffered seq
}

func NewProcessor() *Processor {
	return &Processor{
		buffer: make([]byte, 0),
	}
}

// Add appends a chunk to the internal buffer.
// Returns the accumulated audio data if enough has been collected for ASR.
func (p *Processor) Add(chunk Chunk) ([]byte, bool) {
	if err := chunk.Validate(); err != nil {
		log.Printf("[audio] chunk %d invalid: %v", chunk.Seq, err)
		return nil, false
	}
	if len(chunk.Data) == 0 {
		return nil, false
	}
	p.buffer = append(p.buffer, chunk.Data...)

	// Trigger ASR when we have roughly 2 seconds of 16kHz mono PCM16 audio
	// 16000 samples/sec * 2 bytes/sample * 2 sec = 64000 bytes
	if len(p.buffer) >= 64000 {
		result := make([]byte, len(p.buffer))
		copy(result, p.buffer)
		p.buffer = p.buffer[:0]
		return result, true
	}
	return nil, false
}
