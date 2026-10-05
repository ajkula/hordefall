package music

import "hordefall/internal/audio"

// ===== Types =====

type Song = audio.Song

type Cell = audio.TrackerCell

type Preview = audio.MusicPreview

type Signal = audio.MusicSignal

// ===== Constants =====

const (
	SampleRate         = audio.SampleRate
	NoteEmpty          = audio.NoteEmpty
	NoteOff            = audio.NoteOff
	Channels           = audio.TrackerChannels
	MaximumInstruments = audio.MaximumInstruments
	SignalAlways       = audio.SignalAlways
	SignalHorde        = audio.SignalHorde
	SignalBoss         = audio.SignalBoss
	SignalReactions    = audio.SignalReactions
	SignalDanger       = audio.SignalDanger
	SignalCount        = audio.SignalCount
)

// ===== Public API =====

func Parse(source string) (*Song, error) {
	return audio.ParseSong(source)
}

func NewPreview() *Preview {
	return audio.NewMusicPreview()
}

func ThemeSource() string {
	return audio.ThemeSource()
}
