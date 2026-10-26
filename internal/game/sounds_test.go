package game

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/ajkula/ajkurig/audio"
)

// ===== Constants =====

const soundBytesPerFrame = 8

// ===== Public API =====

func TestEverySoundEffectProducesSound(t *testing.T) {
	for kind := SoundNone + 1; kind < soundKindCount; kind++ {
		peak := renderSoundPeak(kind, audio.SampleRate/2)
		if peak < 0.01 || peak > 2 {
			t.Fatalf("sound %d peak %.3f out of range", kind, peak)
		}
	}
}

func TestEverySoundKindHasItsOwnDefinition(t *testing.T) {
	bank := audio.NewSoundBank()
	ids := RegisterSounds(bank)
	seen := map[audio.SoundID]SoundKind{}
	for kind := SoundNone + 1; kind < soundKindCount; kind++ {
		if previous, isTaken := seen[ids[kind]]; isTaken || ids[kind] == audio.SoundNone {
			t.Fatalf("sound %d shares id %d with sound %d", kind, ids[kind], previous)
		}
		seen[ids[kind]] = kind
	}
}

// ===== Internal =====

func renderSoundPeak(kind SoundKind, frames int) float32 {
	engine, ids := newSilentAudio(1)
	engine.SetMusicOn(false)
	engine.SetEffectsVolume(1)
	engine.Play(ids[kind], 10)
	buffer := make([]byte, frames*soundBytesPerFrame)
	engine.Read(buffer)
	peak := float32(0)
	for offset := 0; offset < len(buffer); offset += 4 {
		peak = max(peak, float32(math.Abs(float64(math.Float32frombits(binary.LittleEndian.Uint32(buffer[offset:]))))))
	}
	return peak
}
