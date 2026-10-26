package music

import (
	_ "embed"

	"github.com/ajkula/ajkurig/audio"
)

// ===== Constants =====

const (
	SignalHorde     = "horde"
	SignalBoss      = "boss"
	SignalReactions = "reactions"
	SignalDanger    = "danger"
	styleSeedStep   = 7919
)

//go:embed theme.trk
var themeSource string

var Styles = []audio.MusicStyle{
	{
		Name: "Neon Pursuit", Tempo: 145, Speed: 5, Tonic: 4, Scale: audio.ScaleHarmonicMinor, Progression: [4]int{0, 5, 6, 4},
		KickRhythm: "x...x...x...x...", SnareRhythm: "....x.......x..x", HihatRhythm: "XxXxXxXxXxXxXxXx",
		BassRhythm: "xxoxxxoxxxoxxfox", ChordRhythm: "x.x.x.x.x.x.x.x.", MelodyRhythm: "x.x.xx.x.x.xx.x.",
		LeadWaveform: audio.WaveSquare, LeadDuty: 0.25, BassWaveform: audio.WaveSaw, ChordDuty: 0.125, BellWaveform: audio.WaveSquare, MelodyOctave: 5,
	},
	{
		Name: "Frozen Wastes", Tempo: 100, Speed: 6, Tonic: 2, Scale: audio.ScaleDorian, Progression: [4]int{0, 3, 0, 6},
		KickRhythm: "x.......x.......", SnareRhythm: "........x.......", HihatRhythm: "X...x...X...x...",
		BassRhythm: "x.......f.......", ChordRhythm: "x.......x.......", MelodyRhythm: "x...x.....x.....",
		LeadWaveform: audio.WaveTriangle, LeadDuty: 0.5, BassWaveform: audio.WaveSine, ChordDuty: 0.5, BellWaveform: audio.WaveTriangle, MelodyOctave: 5,
		BellsDescend: true,
	},
	{
		Name: "Ember March", Tempo: 112, Speed: 6, Tonic: 0, Scale: audio.ScalePhrygian, Progression: [4]int{0, 1, 0, 6},
		KickRhythm: "x..x..x.x..x....", SnareRhythm: "........x.......", HihatRhythm: "X.x.X.x.X.x.X.x.",
		BassRhythm: "x..x..x.x..x..o.", ChordRhythm: "x.....x.....x...", MelodyRhythm: "x.....x.x.......",
		LeadWaveform: audio.WaveSaw, LeadDuty: 0.5, BassWaveform: audio.WaveSquare, ChordDuty: 0.4, BellWaveform: audio.WaveSaw, MelodyOctave: 4,
	},
	{
		Name: "Skyline Rush", Tempo: 155, Speed: 6, Tonic: 5, Scale: audio.ScaleMajor, Progression: [4]int{0, 4, 5, 3},
		KickRhythm: "x...x...x...x...", SnareRhythm: "....x.......x...", HihatRhythm: "xXxXxXxXxXxXxXxX",
		BassRhythm: "x.o.x.o.x.o.x.of", ChordRhythm: "x.xx.x.xx.x.xx.x", MelodyRhythm: "x.xxx.x.x.xxx.x.",
		LeadWaveform: audio.WaveSquare, LeadDuty: 0.125, BassWaveform: audio.WaveSaw, ChordDuty: 0.25, BellWaveform: audio.WaveSquare, MelodyOctave: 5,
	},
	{
		Name: "Grey Transmission", Tempo: 160, Speed: 6, Tonic: 11, Scale: audio.ScaleNaturalMinor, Progression: [4]int{0, 5, 2, 6},
		KickRhythm: "x...x...x...x...", SnareRhythm: "....x.......x...", HihatRhythm: "X.x.X.x.X.x.X.xx",
		BassRhythm: "x.x.o.x.t.x.f.o.", ChordRhythm: "x..x..x...x..x..", MelodyRhythm: "x.......x...x...",
		LeadWaveform: audio.WaveSaw, BassWaveform: audio.WaveSquare, BellWaveform: audio.WaveTriangle, MelodyOctave: 5,
		Overrides: []audio.InstrumentOverride{
			{ID: audio.InstrumentBass, Settings: audio.VoiceSettings{
				Waveform: audio.WaveSquare, Duty: 0.42, Volume: 0.5, Filter: 0.55, Drive: 1.6, SubLevel: 0.6,
				Envelope: audio.Envelope{Attack: 0.002, Decay: 0.12, Sustain: 0.6, Release: 0.04},
			}},
			{ID: audio.InstrumentChords, Settings: audio.VoiceSettings{
				Waveform: audio.WaveSaw, Volume: 0.2, Filter: 0.35, Drive: 3.5, Detune: 0.14,
				Envelope: audio.Envelope{Attack: 0.002, Decay: 0.2, Sustain: 0.55, Release: 0.08},
			}},
			{ID: audio.InstrumentLead, Settings: audio.VoiceSettings{
				Waveform: audio.WaveSaw, Volume: 0.17, Filter: 0.45, Drive: 4, Detune: 0.08,
				Envelope: audio.Envelope{Attack: 0.01, Decay: 0.3, Sustain: 0.75, Release: 0.2},
			}},
		},
	},
}

// ===== Public API =====

func ThemeSource() string {
	return themeSource
}

func Theme() (*audio.Song, error) {
	return audio.ParseSong(themeSource)
}

func Songs(seed uint32) ([]*audio.Song, error) {
	theme, err := Theme()
	if err != nil {
		return nil, err
	}
	songs := make([]*audio.Song, 0, len(Styles)+1)
	songs = append(songs, theme)
	for index := range Styles {
		songs = append(songs, audio.ComposeSong(&Styles[index], theme, seed+uint32(index)*styleSeedStep))
	}
	return songs, nil
}

func SignalNames() []string {
	return []string{SignalHorde, SignalBoss, SignalReactions, SignalDanger}
}
