package main

// ===== Types =====

type InstrumentOverride struct {
	ID       int
	Settings VoiceSettings
}

type MusicStyle struct {
	Name         string
	Tempo        int
	Speed        int
	Tonic        int
	Scale        []int
	Progression  [4]int
	KickRhythm   string
	SnareRhythm  string
	HihatRhythm  string
	BassRhythm   string
	ChordRhythm  string
	MelodyRhythm string
	LeadWaveform Waveform
	LeadDuty     float32
	BassWaveform Waveform
	ChordDuty    float32
	BellWaveform Waveform
	MelodyOctave int
	BellsDescend bool
	Overrides    []InstrumentOverride
}

type songComposer struct {
	style  *MusicStyle
	random Random
	song   *Song
}

// ===== Constants =====

const (
	composedRows        = 16
	composedChordCount  = 4
	melodyLowestDegree  = 0
	melodyHighestDegree = 9
	melodyDropChance    = 0.15
	instrumentKick      = 1
	instrumentSnare     = 2
	instrumentHihat     = 3
	instrumentBass      = 4
	instrumentChords    = 5
	instrumentLead      = 6
	instrumentStab      = 7
	instrumentGrowl     = 8
	instrumentDrone     = 9
	instrumentBell      = 10
	instrumentPulse     = 11
	channelDrums        = 0
	channelBass         = 1
	channelHihat        = 2
	channelChords       = 3
	channelLead         = 4
	channelBossOne      = 5
	channelBossTwo      = 6
	channelBossThree    = 7
	channelBells        = 8
	channelDanger       = 9
	hihatLoudVolume     = 0x30
	hihatSoftVolume     = 0x14
	dangerAccentVolume  = 0x40
	dangerSoftVolume    = 0x1C
	growlAccentVolume   = 0x28
)

var (
	scaleNaturalMinor  = []int{0, 2, 3, 5, 7, 8, 10}
	scaleHarmonicMinor = []int{0, 2, 3, 5, 7, 8, 11}
	scaleDorian        = []int{0, 2, 3, 5, 7, 9, 10}
	scalePhrygian      = []int{0, 1, 3, 5, 7, 8, 10}
	scaleMajor         = []int{0, 2, 4, 5, 7, 9, 11}
)

var musicStyles = []MusicStyle{
	{
		Name: "Neon Pursuit", Tempo: 145, Speed: 5, Tonic: 4, Scale: scaleHarmonicMinor, Progression: [4]int{0, 5, 6, 4},
		KickRhythm: "x...x...x...x...", SnareRhythm: "....x.......x..x", HihatRhythm: "XxXxXxXxXxXxXxXx",
		BassRhythm: "xxoxxxoxxxoxxfox", ChordRhythm: "x.x.x.x.x.x.x.x.", MelodyRhythm: "x.x.xx.x.x.xx.x.",
		LeadWaveform: WaveSquare, LeadDuty: 0.25, BassWaveform: WaveSaw, ChordDuty: 0.125, BellWaveform: WaveSquare, MelodyOctave: 5,
	},
	{
		Name: "Frozen Wastes", Tempo: 100, Speed: 6, Tonic: 2, Scale: scaleDorian, Progression: [4]int{0, 3, 0, 6},
		KickRhythm: "x.......x.......", SnareRhythm: "........x.......", HihatRhythm: "X...x...X...x...",
		BassRhythm: "x.......f.......", ChordRhythm: "x.......x.......", MelodyRhythm: "x...x.....x.....",
		LeadWaveform: WaveTriangle, LeadDuty: 0.5, BassWaveform: WaveSine, ChordDuty: 0.5, BellWaveform: WaveTriangle, MelodyOctave: 5,
		BellsDescend: true,
	},
	{
		Name: "Ember March", Tempo: 112, Speed: 6, Tonic: 0, Scale: scalePhrygian, Progression: [4]int{0, 1, 0, 6},
		KickRhythm: "x..x..x.x..x....", SnareRhythm: "........x.......", HihatRhythm: "X.x.X.x.X.x.X.x.",
		BassRhythm: "x..x..x.x..x..o.", ChordRhythm: "x.....x.....x...", MelodyRhythm: "x.....x.x.......",
		LeadWaveform: WaveSaw, LeadDuty: 0.5, BassWaveform: WaveSquare, ChordDuty: 0.4, BellWaveform: WaveSaw, MelodyOctave: 4,
	},
	{
		Name: "Skyline Rush", Tempo: 155, Speed: 6, Tonic: 5, Scale: scaleMajor, Progression: [4]int{0, 4, 5, 3},
		KickRhythm: "x...x...x...x...", SnareRhythm: "....x.......x...", HihatRhythm: "xXxXxXxXxXxXxXxX",
		BassRhythm: "x.o.x.o.x.o.x.of", ChordRhythm: "x.xx.x.xx.x.xx.x", MelodyRhythm: "x.xxx.x.x.xxx.x.",
		LeadWaveform: WaveSquare, LeadDuty: 0.125, BassWaveform: WaveSaw, ChordDuty: 0.25, BellWaveform: WaveSquare, MelodyOctave: 5,
	},
	{
		Name: "Grey Transmission", Tempo: 160, Speed: 6, Tonic: 11, Scale: scaleNaturalMinor, Progression: [4]int{0, 5, 2, 6},
		KickRhythm: "x...x...x...x...", SnareRhythm: "....x.......x...", HihatRhythm: "X.x.X.x.X.x.X.xx",
		BassRhythm: "x.x.o.x.t.x.f.o.", ChordRhythm: "x..x..x...x..x..", MelodyRhythm: "x.......x...x...",
		LeadWaveform: WaveSaw, BassWaveform: WaveSquare, BellWaveform: WaveTriangle, MelodyOctave: 5,
		Overrides: []InstrumentOverride{
			{ID: instrumentBass, Settings: VoiceSettings{
				Waveform: WaveSquare, Duty: 0.42, Volume: 0.5, Filter: 0.55, Drive: 1.6, SubLevel: 0.35,
				Envelope: Envelope{Attack: 0.002, Decay: 0.12, Sustain: 0.6, Release: 0.04},
			}},
			{ID: instrumentChords, Settings: VoiceSettings{
				Waveform: WaveSaw, Volume: 0.2, Filter: 0.35, Drive: 3.5, Detune: 0.14,
				Envelope: Envelope{Attack: 0.002, Decay: 0.2, Sustain: 0.55, Release: 0.08},
			}},
			{ID: instrumentLead, Settings: VoiceSettings{
				Waveform: WaveSaw, Volume: 0.17, Filter: 0.45, Drive: 4, Detune: 0.08,
				Envelope: Envelope{Attack: 0.01, Decay: 0.3, Sustain: 0.75, Release: 0.2},
			}},
		},
	},
}

// ===== Public API =====

func ComposeSong(style *MusicStyle, template *Song, seed uint32) *Song {
	composer := &songComposer{style: style, random: NewRandom(seed), song: newComposedSong(style, template)}
	for pattern := range composedChordCount * 2 {
		composer.composePattern(pattern)
		composer.song.Order = append(composer.song.Order, pattern)
	}
	return composer.song
}

// ===== Internal =====

func newComposedSong(style *MusicStyle, template *Song) *Song {
	song := &Song{
		Title: style.Name, Tempo: style.Tempo, Speed: style.Speed, TempoBoost: template.TempoBoost,
		Instruments: template.Instruments, ChannelPans: template.ChannelPans, Layers: template.Layers,
		Patterns: map[int]*Pattern{},
	}
	song.Instruments[instrumentLead].Waveform = style.LeadWaveform
	song.Instruments[instrumentLead].Duty = style.LeadDuty
	song.Instruments[instrumentBass].Waveform = style.BassWaveform
	song.Instruments[instrumentChords].Duty = style.ChordDuty
	song.Instruments[instrumentBell].Waveform = style.BellWaveform
	for _, override := range style.Overrides {
		song.Instruments[override.ID] = override.Settings
	}
	return song
}

func (c *songComposer) composePattern(pattern int) {
	chordDegree := c.style.Progression[pattern%composedChordCount]
	rows := make([][trackerChannels]TrackerCell, composedRows)
	for row := range rows {
		rows[row] = emptyRow()
	}
	c.writeDrums(rows)
	c.writeBass(rows, chordDegree)
	c.writeChords(rows, chordDegree)
	c.writeMelody(rows, chordDegree)
	c.writeBossVoices(rows, chordDegree)
	c.writeBells(rows, chordDegree)
	c.writeDanger(rows, chordDegree)
	c.song.Patterns[pattern] = &Pattern{Rows: rows}
}

func emptyRow() [trackerChannels]TrackerCell {
	var row [trackerChannels]TrackerCell
	for channel := range row {
		row[channel] = TrackerCell{Note: noteEmpty}
	}
	return row
}

func (c *songComposer) degreeNote(degree, octave int) int {
	scaleLength := len(c.style.Scale)
	wrappedOctave := octave + floorDivide(degree, scaleLength)
	return wrappedOctave*12 + c.style.Tonic + c.style.Scale[positiveModulo(degree, scaleLength)]
}

func (c *songComposer) chordIntervals(degree int) (int, int) {
	root := c.degreeNote(degree, 4)
	return c.degreeNote(degree+2, 4) - root, c.degreeNote(degree+4, 4) - root
}

func (c *songComposer) writeDrums(rows [][trackerChannels]TrackerCell) {
	for row := range rows {
		isKick := c.style.KickRhythm[row] == 'x'
		isSnare := c.style.SnareRhythm[row] == 'x'
		notes := [3]int{noteEmpty, note("C-3"), note("C-8")}
		instruments := [3]int{0, instrumentKick, instrumentSnare}
		choice := boolToIndex(isKick) + 2*boolToIndex(isSnare && !isKick)
		rows[row][channelDrums] = TrackerCell{Note: notes[choice], Instrument: instruments[choice]}
		accent := c.style.HihatRhythm[row]
		hihatVolumes := map[byte]int{'X': hihatLoudVolume, 'x': hihatSoftVolume}
		volume, hasHihat := hihatVolumes[accent]
		c.setCellIf(rows, row, channelHihat, TrackerCell{Note: note("C-9"), Instrument: instrumentHihat, Effect: effectVolume, Parameter: volume}, hasHihat)
	}
}

func (c *songComposer) writeBass(rows [][trackerChannels]TrackerCell, degree int) {
	third, _ := c.chordIntervals(degree)
	offsets := map[byte]int{'x': 0, 'o': 12, 'f': 7, 't': third + 12}
	root := c.degreeNote(degree, 2)
	for row := range rows {
		offset, hasNote := offsets[c.style.BassRhythm[row]]
		c.setCellIf(rows, row, channelBass, TrackerCell{Note: root + offset, Instrument: instrumentBass}, hasNote)
	}
}

func (c *songComposer) writeChords(rows [][trackerChannels]TrackerCell, degree int) {
	third, fifth := c.chordIntervals(degree)
	chord := TrackerCell{Note: c.degreeNote(degree, 4), Instrument: instrumentChords, Effect: effectArpeggio, Parameter: third<<4 | fifth}
	for row := range rows {
		c.setCellIf(rows, row, channelChords, chord, c.style.ChordRhythm[row] == 'x')
	}
}

func (c *songComposer) writeMelody(rows [][trackerChannels]TrackerCell, degree int) {
	current := degree + 7*boolToIndex(degree < 3)
	for row := range rows {
		isNote := c.style.MelodyRhythm[row] == 'x' && !c.random.Chance(melodyDropChance)
		current = c.nextMelodyDegree(current, degree, row)
		c.setCellIf(rows, row, channelLead, TrackerCell{Note: c.degreeNote(current, c.style.MelodyOctave), Instrument: instrumentLead}, isNote)
	}
}

func (c *songComposer) nextMelodyDegree(current, chordDegree, row int) int {
	steps := [6]int{-2, -1, -1, 1, 1, 2}
	isDownbeat := row%8 == 0
	chordTones := [3]int{chordDegree, chordDegree + 2, chordDegree + 4}
	anchored := nearestDegree(current, chordTones)
	walked := clampInt(current+steps[c.random.Below(len(steps))], melodyLowestDegree, melodyHighestDegree)
	return walked + (anchored-walked)*boolToIndex(isDownbeat)
}

func nearestDegree(current int, candidates [3]int) int {
	best := candidates[0]
	for _, candidate := range candidates {
		best = candidate + (best-candidate)*boolToIndex(absInt(best-current) <= absInt(candidate-current))
	}
	return best
}

func (c *songComposer) writeBossVoices(rows [][trackerChannels]TrackerCell, degree int) {
	root := c.degreeNote(degree, 2)
	for _, row := range []int{0, 3, 6, 10, 12} {
		rows[row][channelBossOne] = TrackerCell{Note: root, Instrument: instrumentStab}
	}
	rows[0][channelBossTwo] = TrackerCell{Note: root - 5, Instrument: instrumentGrowl}
	rows[6][channelBossTwo] = TrackerCell{Note: root - 5, Instrument: instrumentGrowl, Effect: effectVolume, Parameter: growlAccentVolume}
	rows[8][channelBossTwo] = TrackerCell{Note: root - 12, Instrument: instrumentGrowl}
	rows[14][channelBossTwo] = TrackerCell{Note: noteOff}
	drone := [4]int{-12, -11, -12, -14}
	for step, offset := range drone {
		rows[step*4][channelBossThree] = TrackerCell{Note: root + offset, Instrument: instrumentDrone}
	}
}

func (c *songComposer) writeBells(rows [][trackerChannels]TrackerCell, degree int) {
	tones := [4]int{degree, degree + 2, degree + 4, degree + 7}
	for row := 0; row < composedRows; row += 2 {
		step := (row / 2) % 4
		step += (3 - 2*step) * boolToIndex(c.style.BellsDescend)
		rows[row][channelBells] = TrackerCell{Note: c.degreeNote(tones[step], 6), Instrument: instrumentBell}
	}
}

func (c *songComposer) writeDanger(rows [][trackerChannels]TrackerCell, degree int) {
	root := c.degreeNote(degree, 3)
	for row := 0; row < composedRows; row += 2 {
		isAccent := row%8 == 0
		volumes := [2]int{dangerSoftVolume, dangerAccentVolume}
		rows[row][channelDanger] = TrackerCell{Note: root, Instrument: instrumentPulse, Effect: effectVolume, Parameter: volumes[boolToIndex(isAccent)]}
	}
}

func (c *songComposer) setCellIf(rows [][trackerChannels]TrackerCell, row, channel int, cell TrackerCell, shouldSet bool) {
	if !shouldSet {
		return
	}
	rows[row][channel] = cell
}

func note(name string) int {
	value, _ := parseNote(name)
	return value
}

func floorDivide(value, divisor int) int {
	quotient := value / divisor
	return quotient - boolToIndex(value%divisor != 0 && (value < 0) != (divisor < 0))
}

func positiveModulo(value, divisor int) int {
	return ((value % divisor) + divisor) % divisor
}

func absInt(value int) int {
	return max(value, -value)
}
