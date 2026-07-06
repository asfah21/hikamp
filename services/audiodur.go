package services

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/tcolgate/mp3"
)

// GetAudioDuration reads the duration of an audio file in seconds.
// Supports MP3 (via mp3 library) and WAV (via header parsing).
func GetAudioDuration(filePath string) (int, error) {
	lower := strings.ToLower(filePath)
	if strings.HasSuffix(lower, ".mp3") {
		return getMP3Duration(filePath)
	}
	if strings.HasSuffix(lower, ".wav") {
		return getWAVDuration(filePath)
	}
	// For unknown formats, return 0 (unknown)
	return 0, nil
}

// getMP3Duration reads MP3 duration using the tcolgate/mp3 library
func getMP3Duration(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open mp3 file: %w", err)
	}
	defer f.Close()

	d := mp3.NewDecoder(f)
	var frm mp3.Frame
	skipped := 0

	// Count total frames and get the duration
	totalDuration := 0.0
	for {
		if err := d.Decode(&frm, &skipped); err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("failed to decode mp3 frame: %w", err)
		}
		totalDuration += frm.Duration().Seconds()
	}

	return int(math.Round(totalDuration)), nil
}

// getWAVDuration reads WAV duration from its header.
// WAV format: RIFF header (12 bytes) + fmt chunk (24 bytes) + data chunk
// Duration = data_size / (sample_rate * channels * bits_per_sample / 8)
func getWAVDuration(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open wav file: %w", err)
	}
	defer f.Close()

	// Read RIFF header
	header := make([]byte, 44)
	_, err = io.ReadFull(f, header)
	if err != nil {
		return 0, fmt.Errorf("failed to read wav header: %w", err)
	}

	// Check RIFF and WAVE identifiers
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0, fmt.Errorf("invalid WAV file")
	}

	// Parse fmt sub-chunk
	numChannels := int(binary.LittleEndian.Uint16(header[22:24]))
	sampleRate := int(binary.LittleEndian.Uint32(header[24:28]))
	bitsPerSample := int(binary.LittleEndian.Uint16(header[34:36]))

	if sampleRate == 0 || numChannels == 0 || bitsPerSample == 0 {
		return 0, fmt.Errorf("invalid WAV format parameters")
	}

	// Find data chunk size
	// Start searching from byte 36 (after fmt chunk)
	pos := 12
	for pos < 100 { // Search within first 100 bytes
		chunkID := string(header[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(header[pos+4 : pos+8]))
		if chunkID == "data" {
			// Duration = data_size / (sample_rate * channels * bits_per_sample / 8)
			bytesPerSecond := sampleRate * numChannels * bitsPerSample / 8
			if bytesPerSecond == 0 {
				return 0, nil
			}
			duration := float64(chunkSize) / float64(bytesPerSecond)
			return int(math.Round(duration)), nil
		}
		pos += 8 + chunkSize
		if pos+8 > len(header) {
			break
		}
	}

	return 0, fmt.Errorf("could not find data chunk in WAV file")
}
