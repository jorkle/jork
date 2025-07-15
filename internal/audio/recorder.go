package audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/jorkle/jork/internal/models"
)

// Recorder handles audio recording functionality
type Recorder struct {
	stream     *portaudio.Stream
	isRecording bool
	buffer     []float32
	mutex      sync.Mutex
	sampleRate int
	channels   int
}

// NewRecorder creates a new audio recorder
func NewRecorder(sampleRate, channels int) (*Recorder, error) {
	// Initialize PortAudio
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize PortAudio: %w", err)
	}

	recorder := &Recorder{
		sampleRate: sampleRate,
		channels:   channels,
		buffer:     make([]float32, 0),
	}

	return recorder, nil
}

// StartRecording begins recording audio
func (r *Recorder) StartRecording() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isRecording {
		return fmt.Errorf("recording is already in progress")
	}

	// Clear the buffer
	r.buffer = r.buffer[:0]

	// Get default input device
	defaultDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		return fmt.Errorf("failed to get default input device: %w", err)
	}

	// Create input parameters
	inputParams := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   defaultDevice,
			Channels: r.channels,
			Latency:  defaultDevice.DefaultLowInputLatency,
		},
		SampleRate:      float64(r.sampleRate),
		FramesPerBuffer: 1024,
	}

	// Create the stream
	stream, err := portaudio.OpenStream(inputParams, r.recordCallback)
	if err != nil {
		return fmt.Errorf("failed to open audio stream: %w", err)
	}

	r.stream = stream
	r.isRecording = true

	// Start the stream
	if err := r.stream.Start(); err != nil {
		r.isRecording = false
		return fmt.Errorf("failed to start audio stream: %w", err)
	}

	return nil
}

// StopRecording stops recording and returns the recorded audio data
func (r *Recorder) StopRecording() (*models.AudioData, error) {
	r.mutex.Lock()
	if !r.isRecording {
		r.mutex.Unlock()
		return nil, fmt.Errorf("no recording in progress")
	}
	r.isRecording = false
	r.mutex.Unlock()

	if err := r.stream.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop audio stream: %w", err)
	}

	if err := r.stream.Close(); err != nil {
		return nil, fmt.Errorf("failed to close audio stream: %w", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	duration := time.Duration(len(r.buffer)/r.channels) * time.Second / time.Duration(r.sampleRate)

	audioData := &models.AudioData{
		Data:       make([]float32, len(r.buffer)),
		SampleRate: r.sampleRate,
		Duration:   duration,
	}

	copy(audioData.Data, r.buffer)

	return audioData, nil
}

// IsRecording returns true if recording is in progress
func (r *Recorder) IsRecording() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.isRecording
}

// recordCallback is called by PortAudio when audio data is available
func (r *Recorder) recordCallback(inputBuffer []float32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Append the input buffer to our recording buffer
	r.buffer = append(r.buffer, inputBuffer...)
}

// SaveToWAV saves audio data to a WAV file
func (r *Recorder) SaveToWAV(audioData *models.AudioData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create WAV file: %w", err)
	}
	defer file.Close()

	// WAV file header
	header := []byte{
		// RIFF header
		'R', 'I', 'F', 'F',
		0, 0, 0, 0, // File size (will be filled later)
		'W', 'A', 'V', 'E',
		
		// fmt chunk
		'f', 'm', 't', ' ',
		16, 0, 0, 0, // fmt chunk size
		1, 0, // Audio format (1 = PCM)
		0, 0, // Number of channels (will be filled)
		0, 0, 0, 0, // Sample rate (will be filled)
		0, 0, 0, 0, // Byte rate (will be filled)
		0, 0, // Block align (will be filled)
		16, 0, // Bits per sample
		
		// data chunk
		'd', 'a', 't', 'a',
		0, 0, 0, 0, // Data size (will be filled later)
	}

	// Fill in the header values
	channels := uint16(r.channels)
	sampleRate := uint32(audioData.SampleRate)
	bitsPerSample := uint16(16)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := uint32(len(audioData.Data) * 2) // 2 bytes per sample for 16-bit
	fileSize := uint32(len(header)) + dataSize - 8

	// Update header with actual values
	binary.LittleEndian.PutUint32(header[4:8], fileSize)
	binary.LittleEndian.PutUint16(header[22:24], channels)
	binary.LittleEndian.PutUint32(header[24:28], sampleRate)
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	// Write header
	if _, err := file.Write(header); err != nil {
		return fmt.Errorf("failed to write WAV header: %w", err)
	}

	// Convert float32 samples to int16 and write
	for _, sample := range audioData.Data {
		// Convert float32 (-1.0 to 1.0) to int16
		intSample := int16(sample * 32767)
		if err := binary.Write(file, binary.LittleEndian, intSample); err != nil {
			return fmt.Errorf("failed to write audio data: %w", err)
		}
	}

	return nil
}

// Close cleans up the recorder
func (r *Recorder) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isRecording && r.stream != nil {
		r.stream.Stop()
		r.stream.Close()
		r.isRecording = false
	}

	return portaudio.Terminate()
}

