package gui

import (
	"bytes"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"

	"gitlab.com/austin-millan/otaclock/assets"
)

// Audio is decoded to a widely-supported PCM format. 48 kHz stereo is the
// configuration virtually every backend (ALSA, CoreAudio, WASAPI) accepts; the
// 22 kHz source rate triggered "snd_pcm_hw_params: Invalid argument" on some
// ALSA devices.
const (
	soundSampleRate  = 48000
	soundChannels    = 2
	contextReadyWait = 2 * time.Second
)

// SoundPlayer plays the bundled exit.wav for Pomodoro phase-end alerts.
//
// It drives oto directly rather than going through Ebitengine's audio package.
// Ebitengine routes any oto/ALSA failure through its run loop and aborts the
// whole program; by owning the oto context here we can swallow device errors
// and keep the widget running silently.
type SoundPlayer struct {
	mu      sync.Mutex
	ctx     *oto.Context
	pcm     []byte
	current *oto.Player
}

// The only thing sound affects is the Pomodoro phase-end chime, so every
// audio-off log makes that scope explicit.
const soundImpactNote = "the Pomodoro phase-end chime won't play; everything else works normally"

// audioSkipReason returns a human-readable reason to skip opening an audio
// device, or "" when one should be opened. Skipping is no longer load-bearing
// for stability (device errors are contained), but it keeps headless containers
// from spamming ALSA warnings and lets users force silence.
func audioSkipReason() string {
	if os.Getenv("OTA_DISABLE_SOUND") != "" {
		return "disabled via OTA_DISABLE_SOUND"
	}
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/dev/snd"); err != nil {
			return "no audio device found (in Docker, pass --device /dev/snd)"
		}
	}
	return ""
}

// NewSoundPlayer decodes the embedded WAV and opens an audio device. It always
// returns a usable (possibly silent) player and never an error for audio-device
// problems, so the caller can run regardless of the host's audio support.
func NewSoundPlayer() (*SoundPlayer, error) {
	if reason := audioSkipReason(); reason != "" {
		log.Printf("otaclock: sound off (%s) — %s.", reason, soundImpactNote)
		return &SoundPlayer{}, nil
	}

	data, err := assets.Sound("exit")
	if err != nil {
		return nil, err
	}
	stream, err := wav.DecodeWithSampleRate(soundSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   soundSampleRate,
		ChannelCount: soundChannels,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		log.Printf("otaclock: sound off (could not open audio device: %v) — %s.", err, soundImpactNote)
		return &SoundPlayer{}, nil
	}

	select {
	case <-ready:
	case <-time.After(contextReadyWait):
	}

	return &SoundPlayer{ctx: ctx, pcm: pcm}, nil
}

// Play starts the alert sound from the beginning. Any device error is ignored so
// playback can never bring down the application.
func (s *SoundPlayer) Play() {
	if s == nil || s.ctx == nil || len(s.pcm) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	player := s.ctx.NewPlayer(bytes.NewReader(s.pcm))
	player.Play()
	// Hold a reference to the most recent player so it is not collected while
	// it is still feeding the device.
	s.current = player
}
