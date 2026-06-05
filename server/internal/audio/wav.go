package audio

import "encoding/binary"

// WriteWAVHeader prepends a minimal WAV header for PCM16 mono audio.
// sampleRate: e.g. 48000 or 16000
// pcmData: raw PCM16 little-endian samples
func WriteWAVHeader(sampleRate int, pcmData []byte) []byte {
	bitsPerSample := 16
	numChannels := 1
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := len(pcmData)

	// 44-byte WAV header
	header := make([]byte, 44)

	// RIFF header
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize))
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)          // chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)           // PCM format
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))

	// data chunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	// Prepend header to data
	result := make([]byte, 44+dataSize)
	copy(result[:44], header)
	copy(result[44:], pcmData)
	return result
}
