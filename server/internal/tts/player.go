package tts

import (
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

type PCMPlayer struct {
	mu      sync.Mutex
	context *oto.Context
	ready   chan struct{}
}

func NewPCMPlayer() *PCMPlayer {
	return &PCMPlayer{}
}

func (p *PCMPlayer) NewStream() (*AudioStream, error) {
	ctx, err := p.contextForPCM24k()
	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()
	player := ctx.NewPlayer(reader)
	player.SetBufferSize(4096)
	player.Play()

	return &AudioStream{
		reader: reader,
		writer: writer,
		player: player,
	}, nil
}

func (p *PCMPlayer) contextForPCM24k() (*oto.Context, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.context != nil {
		return p.context, nil
	}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   24000,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   80 * time.Millisecond,
	})
	if err != nil {
		return nil, err
	}
	<-ready
	p.context = ctx
	p.ready = ready
	return p.context, nil
}

type AudioStream struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	player *oto.Player
}

func (s *AudioStream) Write(audio []byte) error {
	if s == nil || s.writer == nil || len(audio) == 0 {
		return nil
	}
	_, err := s.writer.Write(audio)
	return err
}

func (s *AudioStream) Close() {
	if s == nil {
		return
	}
	if s.writer != nil {
		_ = s.writer.Close()
	}
	if s.reader != nil {
		_ = s.reader.Close()
	}
	if s.player != nil {
		_ = s.player.Close()
	}
}
