package main

import (
	"math"
	"strings"
	"testing"
)

// ===== Public API =====

func TestThemeSongParses(t *testing.T) {
	song, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	t.Logf("%q: tempo %d, speed %d, %d patterns, order length %d", song.Title, song.Tempo, song.Speed, len(song.Patterns), len(song.Order))
}

func TestSongParserReportsLineOfError(t *testing.T) {
	row := "C-4 01 ... | X-9 01 ..." + strings.Repeat(" | --- .. ...", trackerChannels-2)
	_, err := ParseSong("tempo 120\norder 0\npattern 0\n" + row + "\n")
	if err == nil || !strings.Contains(err.Error(), "X-9") {
		t.Fatalf("invalid note was not reported: %v", err)
	}
	t.Logf("error message: %v", err)
}

func TestMusicIsCleanAtFullIntensity(t *testing.T) {
	tracker := newThemeTracker(t)
	tracker.SetSignals([musicSignalCount]float32{1, 1, maximumSpidersAlive, 1, 1})
	peak, rms := renderStatistics(t, tracker, 16*sampleRate)
	if peak > 1 || rms < 0.02 {
		t.Fatalf("full mix peak %.3f rms %.3f, want peak <= 1 and audible rms", peak, rms)
	}
	t.Logf("full intensity: peak %.3f rms %.3f", peak, rms)
}

func TestLayersFollowSignals(t *testing.T) {
	calm := newThemeTracker(t)
	_, calmRMS := renderStatistics(t, calm, 8*sampleRate)
	intense := newThemeTracker(t)
	intense.SetSignals([musicSignalCount]float32{1, 1, maximumSpidersAlive, 1, 1})
	_, intenseRMS := renderStatistics(t, intense, 8*sampleRate)
	if intenseRMS <= calmRMS*1.2 {
		t.Fatalf("intense mix rms %.3f is not louder than calm mix rms %.3f", intenseRMS, calmRMS)
	}
	t.Logf("calm rms %.3f, intense rms %.3f", calmRMS, intenseRMS)
}

func TestEachSpiderTankAddsADeeperVoice(t *testing.T) {
	song, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	bossChannels := []int{}
	for channel, layer := range song.Layers {
		bossChannels = appendIf(bossChannels, channel, layer.Signal == SignalBoss)
	}
	if len(bossChannels) != maximumSpidersAlive {
		t.Fatalf("%d boss layers, want one per spider tank (%d)", len(bossChannels), maximumSpidersAlive)
	}
	previousNote := 1 << 30
	for spiders, channel := range bossChannels {
		if song.Layers[channel].Threshold != float32(spiders+1) {
			t.Fatalf("boss layer on channel %d starts at %.0f tanks, want %d", channel+1, song.Layers[channel].Threshold, spiders+1)
		}
		firstNote := song.Patterns[song.Order[0]].Rows[0][channel].Note
		if firstNote >= previousNote {
			t.Fatalf("boss voice %d (note %d) is not deeper than the previous one (note %d)", spiders+1, firstNote, previousNote)
		}
		previousNote = firstNote
	}
}

func TestEverySoundEffectProducesSound(t *testing.T) {
	for kind := SoundPop; kind < soundKindCount; kind++ {
		engine := &AudioEngine{random: NewRandom(1), EffectsVolume: 1}
		engine.startSound(&soundTable[kind])
		peak := float32(0)
		for range sampleRate / 2 {
			left, right := engine.renderEffects()
			peak = max(peak, abs(left), abs(right))
		}
		if peak < 0.01 || peak > 2 {
			t.Fatalf("sound %d peak %.3f out of range", kind, peak)
		}
	}
}

// ===== Internal =====

func newThemeTracker(t *testing.T) *Tracker {
	song, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	return NewTracker(song)
}

func renderStatistics(t *testing.T, tracker *Tracker, samples int) (float32, float32) {
	peak, sumSquares := float32(0), float64(0)
	for range samples {
		left, right := tracker.Render()
		mixedLeft, mixedRight := softClip(left*0.5), softClip(right*0.5)
		if math.IsNaN(float64(mixedLeft)) || math.IsNaN(float64(mixedRight)) {
			t.Fatalf("NaN in music output")
		}
		peak = max(peak, abs(mixedLeft), abs(mixedRight))
		sumSquares += float64(mixedLeft*mixedLeft + mixedRight*mixedRight)
	}
	return peak, float32(math.Sqrt(sumSquares / float64(2*samples)))
}
