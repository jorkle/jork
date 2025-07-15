package audio

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/jorkle/jork/internal/models"
)

// Player handles audio playback functionality
type Player struct {
	isPlaying  bool
	mutex      sync.RWMutex
	currentCmd *exec.Cmd
}

// NewPlayer creates a new audio player
func NewPlayer() *Player {
	return &Player{
		isPlaying: false,
	}
}

// PlayAudioData plays audio data directly
func (p *Player) PlayAudioData(audioData *models.AudioData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isPlaying {
		return fmt.Errorf("audio is already playing")
	}

	// Create a temporary WAV file
	tempFile, err := os.CreateTemp("", "jork_audio_*.wav")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Save audio data to temporary WAV file
	recorder := &Recorder{
		sampleRate: audioData.SampleRate,
		channels:   1, // Assuming mono for simplicity
	}
	
	if err := recorder.SaveToWAV(audioData, tempFile.Name()); err != nil {
		return fmt.Errorf("failed to save audio data: %w", err)
	}

	// Play the WAV file
	return p.PlayFile(tempFile.Name())
}

// PlayFile plays an audio file using the system's default audio player
func (p *Player) PlayFile(filename string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isPlaying {
		return fmt.Errorf("audio is already playing")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s", filename)
	}

	// Try different audio players based on what's available
	var cmd *exec.Cmd
	
	// Try aplay (ALSA) first - common on Linux
	if _, err := exec.LookPath("aplay"); err == nil {
		cmd = exec.Command("aplay", filename)
	} else if _, err := exec.LookPath("paplay"); err == nil {
		// Try paplay (PulseAudio)
		cmd = exec.Command("paplay", filename)
	} else if _, err := exec.LookPath("ffplay"); err == nil {
		// Try ffplay (FFmpeg) - more universal but requires FFmpeg
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", filename)
	} else {
		return fmt.Errorf("no suitable audio player found (tried: aplay, paplay, ffplay)")
	}

	p.currentCmd = cmd
	p.isPlaying = true

	// Start the command in a goroutine
	go func() {
		defer func() {
			p.mutex.Lock()
			p.isPlaying = false
			p.currentCmd = nil
			p.mutex.Unlock()
		}()

		if err := cmd.Run(); err != nil {
			// Log error but don't return it since we're in a goroutine
			fmt.Printf("Error playing audio: %v\n", err)
		}
	}()

	return nil
}

// PlayMP3File plays an MP3 file (for OpenAI TTS output)
func (p *Player) PlayMP3File(filename string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isPlaying {
		return fmt.Errorf("audio is already playing")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s", filename)
	}

	// Try different MP3 players
	var cmd *exec.Cmd
	
	if _, err := exec.LookPath("mpg123"); err == nil {
		cmd = exec.Command("mpg123", filename)
	} else if _, err := exec.LookPath("ffplay"); err == nil {
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", filename)
	} else if _, err := exec.LookPath("paplay"); err == nil {
		// Convert MP3 to WAV using ffmpeg and play with paplay
		return p.playMP3WithFFmpeg(filename)
	} else {
		return fmt.Errorf("no suitable MP3 player found (tried: mpg123, ffplay, paplay+ffmpeg)")
	}

	p.currentCmd = cmd
	p.isPlaying = true

	// Start the command in a goroutine
	go func() {
		defer func() {
			p.mutex.Lock()
			p.isPlaying = false
			p.currentCmd = nil
			p.mutex.Unlock()
		}()

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error playing MP3: %v\n", err)
		}
	}()

	return nil
}

// playMP3WithFFmpeg converts MP3 to WAV and plays it
func (p *Player) playMP3WithFFmpeg(filename string) error {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found, cannot convert MP3")
	}

	// Create temporary WAV file
	tempWAV, err := os.CreateTemp("", "jork_converted_*.wav")
	if err != nil {
		return fmt.Errorf("failed to create temporary WAV file: %w", err)
	}
	defer os.Remove(tempWAV.Name())
	tempWAV.Close()

	// Convert MP3 to WAV
	convertCmd := exec.Command("ffmpeg", "-i", filename, "-y", tempWAV.Name())
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("failed to convert MP3 to WAV: %w", err)
	}

	// Play the converted WAV file
	return p.PlayFile(tempWAV.Name())
}

// StopPlayback stops the current audio playback
func (p *Player) StopPlayback() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isPlaying || p.currentCmd == nil {
		return fmt.Errorf("no audio is currently playing")
	}

	// Kill the current command
	if err := p.currentCmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop playback: %w", err)
	}

	p.isPlaying = false
	p.currentCmd = nil

	return nil
}

// IsPlaying returns true if audio is currently playing
func (p *Player) IsPlaying() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.isPlaying
}

// WaitForPlayback waits for the current playback to finish
func (p *Player) WaitForPlayback() {
	for p.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
}

// GetSupportedFormats returns the audio formats supported by the system
func (p *Player) GetSupportedFormats() []string {
	formats := []string{}
	
	// Check for WAV support
	if _, err := exec.LookPath("aplay"); err == nil {
		formats = append(formats, "WAV (via aplay)")
	}
	if _, err := exec.LookPath("paplay"); err == nil {
		formats = append(formats, "WAV (via paplay)")
	}
	
	// Check for MP3 support
	if _, err := exec.LookPath("mpg123"); err == nil {
		formats = append(formats, "MP3 (via mpg123)")
	}
	
	// Check for universal support via ffplay
	if _, err := exec.LookPath("ffplay"); err == nil {
		formats = append(formats, "Multiple formats (via ffplay)")
	}
	
	return formats
}

// StreamAudioFromReader plays audio data from an io.Reader (useful for streaming)
func (p *Player) StreamAudioFromReader(reader io.Reader, format string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isPlaying {
		return fmt.Errorf("audio is already playing")
	}

	// Create temporary file with appropriate extension
	var tempFile *os.File
	var err error
	
	switch format {
	case "mp3":
		tempFile, err = os.CreateTemp("", "jork_stream_*.mp3")
	case "wav":
		tempFile, err = os.CreateTemp("", "jork_stream_*.wav")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	defer os.Remove(tempFile.Name())

	// Copy data from reader to temporary file
	if _, err := io.Copy(tempFile, reader); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write audio data: %w", err)
	}
	tempFile.Close()

	// Play the temporary file
	switch format {
	case "mp3":
		return p.PlayMP3File(tempFile.Name())
	case "wav":
		return p.PlayFile(tempFile.Name())
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

