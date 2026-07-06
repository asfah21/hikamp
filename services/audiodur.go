package services

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tcolgate/mp3"
)

// AudioMetadata holds the metadata extracted by ffprobe
type AudioMetadata struct {
	Duration    int    // in seconds
	DurationStr string // formatted as mm:ss
	Bitrate     int    // in kbps
	SampleRate  int    // in Hz
}

// ffprobeOutput represents the JSON output structure from ffprobe
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
		BitRate    string `json:"bit_rate"`
	} `json:"streams"`
}

// GetAudioMetadata reads audio metadata using ffprobe.
// Returns duration (seconds), duration string (mm:ss), bitrate (kbps), and sample rate (Hz).
func GetAudioMetadata(filePath string) (*AudioMetadata, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var info ffprobeOutput
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	meta := &AudioMetadata{}

	// Parse duration
	if info.Format.Duration != "" {
		durationSec, err := strconv.ParseFloat(info.Format.Duration, 64)
		if err == nil {
			meta.Duration = int(durationSec + 0.5) // round to nearest second
			minutes := meta.Duration / 60
			seconds := meta.Duration % 60
			meta.DurationStr = fmt.Sprintf("%d:%02d", minutes, seconds)
		}
	}

	// Parse bitrate from format (overall bitrate in bps -> kbps)
	if info.Format.BitRate != "" {
		bitrateBps, err := strconv.Atoi(info.Format.BitRate)
		if err == nil {
			meta.Bitrate = bitrateBps / 1000
		}
	}

	// Parse sample rate from the first audio stream
	for _, stream := range info.Streams {
		if stream.CodecType == "audio" {
			if stream.SampleRate != "" {
				sr, err := strconv.Atoi(stream.SampleRate)
				if err == nil {
					meta.SampleRate = sr
				}
			}
			// Use stream bitrate if format bitrate is not available
			if meta.Bitrate == 0 && stream.BitRate != "" {
				bitrateBps, err := strconv.Atoi(stream.BitRate)
				if err == nil {
					meta.Bitrate = bitrateBps / 1000
				}
			}
			break
		}
	}

	return meta, nil
}

// GetAudioDuration reads the duration of an audio file in seconds.
// Supports MP3 (via mp3 library) and WAV (via header parsing).
// Falls back to ffprobe if available, otherwise uses the legacy methods.
func GetAudioDuration(filePath string) (int, error) {
	// Try ffprobe first
	meta, err := GetAudioMetadata(filePath)
	if err == nil && meta.Duration > 0 {
		return meta.Duration, nil
	}

	// Fallback to legacy methods
	lower := strings.ToLower(filePath)
	if strings.HasSuffix(lower, ".mp3") {
		return getMP3Duration(filePath)
	}
	if strings.HasSuffix(lower, ".wav") {
		return getWAVDuration(filePath)
	}
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
