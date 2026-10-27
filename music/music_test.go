package music

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajkula/ajkurig/audio"
)

// ===== Constants =====

const (
	bossSignalMaximum = 3
	bytesPerFrame     = 8
	testSeed          = 1234
)

// ===== Public API =====

func TestThemeSongParses(t *testing.T) {
	song := mustTheme(t)
	t.Logf("%q: tempo %d, speed %d, %d patterns, order length %d", song.Title, song.Tempo, song.Speed, len(song.Patterns), len(song.Order))
}

func TestMusicIsCleanAtFullIntensity(t *testing.T) {
	peak, rms := renderSong(t, mustTheme(t), fullSignals(), 16*audio.SampleRate)
	if peak > 1 || rms < 0.02 {
		t.Fatalf("full mix peak %.3f rms %.3f, want peak <= 1 and audible rms", peak, rms)
	}
	t.Logf("full intensity: peak %.3f rms %.3f", peak, rms)
}

func TestLayersFollowSignals(t *testing.T) {
	_, calmRMS := renderSong(t, mustTheme(t), map[string]float32{}, 8*audio.SampleRate)
	_, intenseRMS := renderSong(t, mustTheme(t), fullSignals(), 8*audio.SampleRate)
	if intenseRMS <= calmRMS*1.2 {
		t.Fatalf("intense mix rms %.3f is not louder than calm mix rms %.3f", intenseRMS, calmRMS)
	}
	t.Logf("calm rms %.3f, intense rms %.3f", calmRMS, intenseRMS)
}

func TestEachSpiderTankAddsADeeperVoice(t *testing.T) {
	song := mustTheme(t)
	bossChannels := []int{}
	for channel, layer := range song.Layers {
		if layer.SignalName == SignalBoss.Name() {
			bossChannels = append(bossChannels, channel)
		}
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

func TestComposedStylesAreValidCleanAndDistinct(t *testing.T) {
	theme := mustTheme(t)
	signatures := map[string]bool{}
	for index := range Styles {
		song := audio.ComposeSong(&Styles[index], theme, testSeed)
		if err := audio.ValidateSong(song); err != nil {
			t.Fatalf("%s is invalid: %v", song.Title, err)
		}
		peak, rms := renderSong(t, song, fullSignals(), 6*audio.SampleRate)
		if peak > 1 || rms < 0.02 {
			t.Fatalf("%s: peak %.3f rms %.3f", song.Title, peak, rms)
		}
		firstLeadNote := song.Patterns[0].Rows[0][audio.ChannelChords].Note
		signatures[fmt.Sprintf("%d/%d/%d", song.Tempo, firstLeadNote, song.Instruments[audio.InstrumentLead].Waveform)] = true
		t.Logf("%-18s tempo %d  peak %.2f  rms %.3f", song.Title, song.Tempo, peak, rms)
	}
	if len(signatures) != len(Styles) {
		t.Fatalf("styles are not distinct: %v", signatures)
	}
}

func TestMusicIsBalancedAndWideInStereo(t *testing.T) {
	for _, song := range mustSongs(t) {
		tracker := newTracker(song, fullSignals())
		room := &audio.StereoRoom{}
		lowLeft, lowRight := float32(0), float32(0)
		energyLeft, energyRight, energySide, energyMid := float64(0), float64(0), float64(0), float64(0)
		bassLeft, bassRight := float64(0), float64(0)
		for range 12 * audio.SampleRate {
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
		balance, bassBalance, width := energyLeft/energyRight, bassLeft/bassRight, energySide/energyMid
		t.Logf("%-18s balance L/R %.2f, bass L/R %.2f, side/mid %.3f", song.Title, balance, bassBalance, width)
		if balance < 0.8 || balance > 1.25 || bassBalance < 0.9 || bassBalance > 1.1 || width < 0.01 {
			t.Errorf("%s is lopsided or flat: balance %.2f, bass %.2f, width %.3f", song.Title, balance, bassBalance, width)
		}
	}
}

func TestExtraLayersPlayChordTonesInEverySong(t *testing.T) {
	tonalChannels := []int{audio.ChannelPad, audio.ChannelPluck, audio.ChannelOffbeat, audio.ChannelHarmony}
	for _, song := range mustSongs(t) {
		notesPerChannel := [audio.TrackerChannels]int{}
		for _, pattern := range song.Patterns {
			chords := audio.DetectChords(pattern.Rows)
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
		t.Logf("%-18s notes per channel %v", song.Title, notesPerChannel)
	}
}

func TestTrackerPreviewFollowsNumberedSignals(t *testing.T) {
	preview := NewPreview()
	for signal := range SignalCount {
		preview.SetSignal(signal, float32(signal)+0.5)
	}
	for signal := SignalHorde; signal < SignalCount; signal++ {
		if value := preview.SignalValue(signal.Name()); value != float32(signal)+0.5 {
			t.Fatalf("signal %q reads %.1f, want %.1f", signal.Name(), value, float32(signal)+0.5)
		}
	}
	calmPeak, calmRMS := renderPreview(t, 0)
	intensePeak, intenseRMS := renderPreview(t, bossSignalMaximum)
	if intenseRMS <= calmRMS*1.05 || max(calmPeak, intensePeak) >= 1 || calmRMS <= 0 {
		t.Fatalf("preview calm rms %.4f peak %.3f, boss rms %.4f peak %.3f", calmRMS, calmPeak, intenseRMS, intensePeak)
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
	for index, song := range mustSongs(t) {
		path := filepath.Join(outputDirectory, fmt.Sprintf("%d_%s.wav", index+1, strings.ReplaceAll(song.Title, " ", "_")))
		if err := audio.WriteWAV(path, newTracker(song, fullSignals()), 30*audio.SampleRate); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
	}
}

// ===== Internal =====

func mustTheme(t *testing.T) *audio.Song {
	song, err := Theme()
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	return song
}

func mustSongs(t *testing.T) []*audio.Song {
	songs, err := Songs(testSeed)
	if err != nil {
		t.Fatalf("songs do not compose: %v", err)
	}
	return songs
}

func fullSignals() map[string]float32 {
	return map[string]float32{SignalHorde.Name(): 1, SignalBoss.Name(): bossSignalMaximum, SignalReactions.Name(): 1, SignalDanger.Name(): 1}
}

func newBoard(signals map[string]float32) *audio.SignalBoard {
	board := audio.NewSignalBoard(SignalNames()...)
	for name, value := range signals {
		board.Set(board.Register(name), value)
	}
	return board
}

func newTracker(song *audio.Song, signals map[string]float32) *audio.Tracker {
	board := newBoard(signals)
	tracker := audio.NewTracker(song, board)
	tracker.SetSignals(board.Values())
	return tracker
}

func renderSong(t *testing.T, song *audio.Song, signals map[string]float32, samples int) (float32, float32) {
	board := newBoard(signals)
	engine := audio.NewSilentEngine(audio.NewSoundBank(), board)
	engine.SetSongs([]*audio.Song{song})
	engine.PushSignals()
	buffer := make([]byte, samples*bytesPerFrame)
	engine.Read(buffer)
	peak, sumSquares := float32(0), float64(0)
	for offset := 0; offset < len(buffer); offset += 4 {
		value := math.Float32frombits(binary.LittleEndian.Uint32(buffer[offset:]))
		if math.IsNaN(float64(value)) {
			t.Fatalf("NaN in %s", song.Title)
		}
		peak, sumSquares = max(peak, float32(math.Abs(float64(value)))), sumSquares+float64(value*value)
	}
	return peak, float32(math.Sqrt(sumSquares / float64(len(buffer)/4)))
}

func renderPreview(t *testing.T, boss float32) (float32, float32) {
	preview := NewPreview()
	preview.SetSong(mustTheme(t))
	for signal := range SignalCount {
		preview.SetSignal(signal, 0)
	}
	preview.SetSignal(SignalHorde, 1)
	preview.SetSignal(SignalBoss, boss)
	preview.Play(0, 0)
	buffer := make([]byte, 8*SampleRate*bytesPerFrame)
	preview.Read(buffer)
	peak, sumSquares := float32(0), float64(0)
	for offset := 0; offset < len(buffer); offset += 4 {
		value := math.Float32frombits(binary.LittleEndian.Uint32(buffer[offset:]))
		peak, sumSquares = max(peak, float32(math.Abs(float64(value)))), sumSquares+float64(value*value)
	}
	return peak, float32(math.Sqrt(sumSquares / float64(len(buffer)/4)))
}

func boolToIndex(value bool) int {
	if value {
		return 1
	}
	return 0
}
