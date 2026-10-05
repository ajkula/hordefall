package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hordefall/internal/rng"
)

// ===== Constants =====

const bossSignalMaximum = 3

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
	tracker.SetSignals([SignalCount]float32{1, 1, bossSignalMaximum, 1, 1})
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
	intense.SetSignals([SignalCount]float32{1, 1, bossSignalMaximum, 1, 1})
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
	if len(bossChannels) != bossSignalMaximum {
		t.Fatalf("%d boss layers, want one per spider tank (%d)", len(bossChannels), bossSignalMaximum)
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
		engine := &Engine{random: rng.New(1), EffectsVolume: 1}
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

func TestMusicToggleKeepsSoundEffects(t *testing.T) {
	engine := &Engine{random: rng.New(1), EffectsVolume: 1, MusicVolume: 1, tracker: newThemeTracker(t)}
	engine.SetMusicOn(false)
	if engine.IsMusicOn() {
		t.Fatalf("music still on after toggle")
	}
	engine.startSound(&soundTable[SoundPop])
	peak := float32(0)
	for range sampleRate / 10 {
		left, right := engine.renderFrame()
		peak = max(peak, abs(left), abs(right))
	}
	if peak < 0.01 {
		t.Fatalf("sound effects silenced by the music toggle (peak %.3f)", peak)
	}
}

func TestComposedStylesAreValidCleanAndDistinct(t *testing.T) {
	theme, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	signatures := map[string]bool{}
	for index := range musicStyles {
		song := ComposeSong(&musicStyles[index], theme, 1234)
		if err := validateSong(song); err != nil {
			t.Fatalf("%s is invalid: %v", song.Title, err)
		}
		tracker := NewTracker(song)
		tracker.SetSignals([SignalCount]float32{1, 1, bossSignalMaximum, 1, 1})
		peak, rms := renderStatistics(t, tracker, 6*sampleRate)
		if peak > 1 || rms < 0.02 {
			t.Fatalf("%s: peak %.3f rms %.3f", song.Title, peak, rms)
		}
		firstLeadNote := song.Patterns[0].Rows[0][channelChords].Note
		signature := fmt.Sprintf("%d/%d/%d", song.Tempo, firstLeadNote, song.Instruments[instrumentLead].Waveform)
		signatures[signature] = true
		t.Logf("%-14s tempo %d  peak %.2f  rms %.3f", song.Title, song.Tempo, peak, rms)
	}
	if len(signatures) != len(musicStyles) {
		t.Fatalf("styles are not distinct: %v", signatures)
	}
}

func TestExportSongPreviews(t *testing.T) {
	outputDirectory := os.Getenv("HORDEFALL_WAV_DIR")
	if outputDirectory == "" {
		t.Skip("HORDEFALL_WAV_DIR not set")
	}
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	engine := &Engine{random: rng.New(1)}
	engine.attachSongs(newThemeTracker(t).song, uint32(time.Now().UnixNano()))
	for index, song := range engine.songs {
		tracker := NewTracker(song)
		tracker.SetSignals([SignalCount]float32{1, 1, bossSignalMaximum, 1, 1})
		path := filepath.Join(outputDirectory, fmt.Sprintf("%d_%s.wav", index+1, strings.ReplaceAll(song.Title, " ", "_")))
		writePreview(t, tracker, path, 30*sampleRate)
		t.Logf("wrote %s", path)
	}
}

func writePreview(t *testing.T, tracker *Tracker, path string, samples int) {
	pcm := make([]byte, 0, samples*4)
	for range samples {
		left, right := tracker.Render()
		pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(softClip(left*0.5)*32000)))
		pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(softClip(right*0.5)*32000)))
	}
	header := make([]byte, 0, 44)
	header = append(header, "RIFF"...)
	header = binary.LittleEndian.AppendUint32(header, uint32(36+len(pcm)))
	header = append(header, "WAVEfmt "...)
	header = binary.LittleEndian.AppendUint32(header, 16)
	header = binary.LittleEndian.AppendUint16(header, 1)
	header = binary.LittleEndian.AppendUint16(header, 2)
	header = binary.LittleEndian.AppendUint32(header, sampleRate)
	header = binary.LittleEndian.AppendUint32(header, sampleRate*4)
	header = binary.LittleEndian.AppendUint16(header, 4)
	header = binary.LittleEndian.AppendUint16(header, 16)
	header = append(header, "data"...)
	header = binary.LittleEndian.AppendUint32(header, uint32(len(pcm)))
	if err := os.WriteFile(path, append(header, pcm...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyPlaylistSilencesMusicButNotEffects(t *testing.T) {
	engine := &Engine{random: rng.New(1), EffectsVolume: 1, MusicVolume: 1}
	engine.attachSongs(newThemeTracker(t).song, 3)
	engine.SetPlaylistEmpty(true)
	musicPeak := float32(0)
	for range sampleRate {
		left, right := engine.renderFrame()
		musicPeak = max(musicPeak, abs(left), abs(right))
	}
	engine.startSound(&soundTable[SoundPop])
	effectPeak := float32(0)
	for range sampleRate / 10 {
		left, right := engine.renderFrame()
		effectPeak = max(effectPeak, abs(left), abs(right))
	}
	if musicPeak > 0.0001 || effectPeak < 0.01 {
		t.Fatalf("empty playlist: music peak %.4f (want silence), effects peak %.3f (want audible)", musicPeak, effectPeak)
	}
}

func TestMusicIsBalancedAndWideInStereo(t *testing.T) {
	theme, err := ParseSong(themeSource)
	if err != nil {
		t.Fatal(err)
	}
	songs := []*Song{theme}
	for index := range musicStyles {
		songs = append(songs, ComposeSong(&musicStyles[index], theme, uint32(index+1)*7919))
	}
	for _, song := range songs {
		tracker := NewTracker(song)
		tracker.SetSignals([SignalCount]float32{1, 1, bossSignalMaximum, 1, 1})
		room := &StereoRoom{}
		lowLeft, lowRight := float32(0), float32(0)
		energyLeft, energyRight, energySide, energyMid := float64(0), float64(0), float64(0), float64(0)
		bassLeft, bassRight := float64(0), float64(0)
		for range 12 * sampleRate {
			left, right := room.Process(tracker.Render())
			lowLeft += (left - lowLeft) * 0.02
			lowRight += (right - lowRight) * 0.02
			energyLeft += float64(left * left)
			energyRight += float64(right * right)
			bassLeft += float64(lowLeft * lowLeft)
			bassRight += float64(lowRight * lowRight)
			energyMid += float64((left + right) * (left + right))
			energySide += float64((left - right) * (left - right))
		}
		balance := energyLeft / energyRight
		bassBalance := bassLeft / bassRight
		width := energySide / energyMid
		t.Logf("%-18s balance L/R %.2f, bass L/R %.2f, side/mid %.3f", song.Title, balance, bassBalance, width)
		if balance < 0.8 || balance > 1.25 || bassBalance < 0.9 || bassBalance > 1.1 || width < 0.01 {
			t.Errorf("%s is lopsided or flat: balance %.2f, bass %.2f, width %.3f", song.Title, balance, bassBalance, width)
		}
	}
}

func TestExtraLayersPlayChordTonesInEverySong(t *testing.T) {
	theme, err := ParseSong(themeSource)
	if err != nil {
		t.Fatal(err)
	}
	songs := []*Song{theme}
	for index := range musicStyles {
		songs = append(songs, ComposeSong(&musicStyles[index], theme, uint32(index+1)*7919))
	}
	tonalChannels := []int{channelPad, channelPluck, channelOffbeat, channelHarmony}
	for _, song := range songs {
		notesPerChannel := [trackerChannels]int{}
		for _, pattern := range song.Patterns {
			chords := detectChords(pattern.Rows)
			for rowIndex, row := range pattern.Rows {
				for channel, cell := range row {
					notesPerChannel[channel] += boolToIndex(cell.Note >= 0)
				}
				for _, channel := range tonalChannels {
					cell := row[channel]
					if cell.Note >= 0 && !chords[rowIndex].Contains(cell.Note) {
						t.Fatalf("%s: channel %d plays %d outside chord %+v", song.Title, channel+1, cell.Note, chords[rowIndex])
					}
				}
			}
		}
		for channel, count := range notesPerChannel {
			if count == 0 {
				t.Errorf("%s: channel %d never plays", song.Title, channel+1)
			}
		}
		tracker := NewTracker(song)
		tracker.SetSignals([SignalCount]float32{1, 1, bossSignalMaximum, 1, 1})
		peak, rms := renderStatistics(t, tracker, 6*sampleRate)
		t.Logf("%-18s notes per channel %v  peak %.2f rms %.3f", song.Title, notesPerChannel, peak, rms)
	}
}

func appendIf[T any](items []T, item T, shouldAppend bool) []T {
	if !shouldAppend {
		return items
	}
	return append(items, item)
}

func TestSongCrossfadeStaysClean(t *testing.T) {
	engine := NewSilentEngine(99)
	engine.MusicVolume = 1
	for index := range engine.SongCount() + 1 {
		engine.SelectSong(index % engine.SongCount())
		for range songCrossfadeSeconds * sampleRate / 4 {
			left, right := engine.renderMusic()
			if math.IsNaN(float64(left)) || math.IsNaN(float64(right)) || abs(left) > 4 {
				t.Fatalf("bad sample while crossfading into song %d", index)
			}
		}
	}
}

func TestMusicPreviewPlaysSeeksAndAuditions(t *testing.T) {
	song, err := ParseTheme()
	if err != nil {
		t.Fatal(err)
	}
	preview := NewMusicPreview()
	preview.SetSong(song)
	buffer := make([]byte, bytesPerFrame*4096)
	preview.Read(buffer)
	if peakOf(buffer) != 0 {
		t.Fatal("a stopped preview is not silent")
	}
	preview.Play(1, 3)
	if orderIndex, row := preview.Position(); orderIndex != 1 || row != 3 {
		t.Fatalf("play started at %d/%d, want 1/3", orderIndex, row)
	}
	preview.Read(buffer)
	if peakOf(buffer) == 0 {
		t.Fatal("a playing preview is silent")
	}
	preview.Stop()
	preview.Audition(1, 48, 0.5)
	preview.Read(buffer)
	if peakOf(buffer) == 0 {
		t.Fatal("audition is silent")
	}
}

func peakOf(buffer []byte) float32 {
	peak := float32(0)
	for offset := 0; offset+4 <= len(buffer); offset += 4 {
		peak = max(peak, abs(math.Float32frombits(binary.LittleEndian.Uint32(buffer[offset:]))))
	}
	return peak
}
