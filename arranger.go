package main

// ===== Types =====

type ChordTones struct {
	Root  int
	Third int
	Fifth int
}

type layerArranger func(rows [][trackerChannels]TrackerCell, chords []ChordTones, song *Song)

// ===== Constants =====

const (
	channelShaker      = 10
	channelClap        = 11
	channelPad         = 12
	channelPluck       = 13
	channelOffbeat     = 14
	channelHarmony     = 15
	instrumentShaker   = 12
	instrumentClap     = 13
	instrumentPad      = 14
	instrumentPluck    = 15
	instrumentOffbeat  = 16
	instrumentHarmony  = 17
	slowSongTempo      = 115
	shakerAccentVolume = 0x30
	shakerSoftVolume   = 0x18
	defaultMinorThird  = 3
	defaultFifth       = 7
	harmonyLowestDrop  = 9
	padOctave          = 4
	pluckOctave        = 5
	offbeatOctave      = 4
	backbeatSpacing    = 8
	backbeatOffset     = 4
)

var extraLayerArrangers = []struct {
	Channel int
	Arrange layerArranger
}{
	{channelShaker, arrangeShaker},
	{channelClap, arrangeClap},
	{channelPad, arrangePad},
	{channelPluck, arrangePluck},
	{channelOffbeat, arrangeOffbeat},
	{channelHarmony, arrangeHarmony},
}

var pluckChordSteps = [4]int{0, 1, 2, 1}

// ===== Public API =====

func ArrangeExtraLayers(song *Song) {
	for _, pattern := range song.Patterns {
		chords := detectChords(pattern.Rows)
		for _, arranger := range extraLayerArrangers {
			arrangeIfEmpty(pattern.Rows, chords, song, arranger.Channel, arranger.Arrange)
		}
	}
}

func (c ChordTones) Tone(step, octave int) int {
	offsets := [3]int{0, c.Third, c.Fifth}
	return octave*12 + positiveModulo(c.Root, 12) + offsets[positiveModulo(step, 3)] + 12*floorDivide(step, 3)
}

// ===== Internal =====

func arrangeIfEmpty(rows [][trackerChannels]TrackerCell, chords []ChordTones, song *Song, channel int, arrange layerArranger) {
	for _, row := range rows {
		if row[channel].Note != noteEmpty {
			return
		}
	}
	arrange(rows, chords, song)
}

func detectChords(rows [][trackerChannels]TrackerCell) []ChordTones {
	chords := make([]ChordTones, len(rows))
	current := firstChord(rows)
	for index, row := range rows {
		current = chordFromCell(row[channelChords], current)
		chords[index] = current
	}
	return chords
}

func firstChord(rows [][trackerChannels]TrackerCell) ChordTones {
	fallback := ChordTones{Third: defaultMinorThird, Fifth: defaultFifth}
	for _, row := range rows {
		fallback.Root += (row[channelBass].Note - fallback.Root) * boolToIndex(row[channelBass].Note >= 0 && fallback.Root == 0)
	}
	for _, row := range rows {
		chord := chordFromCell(row[channelChords], fallback)
		if chord != fallback {
			return chord
		}
	}
	return fallback
}

func chordFromCell(cell TrackerCell, current ChordTones) ChordTones {
	isChord := cell.Note >= 0 && cell.Effect == effectArpeggio
	if !isChord {
		return current
	}
	return ChordTones{Root: cell.Note, Third: cell.Parameter >> 4, Fifth: cell.Parameter & 0xF}
}

func arrangeShaker(rows [][trackerChannels]TrackerCell, _ []ChordTones, _ *Song) {
	quietByParity := [2]int{}
	for index, row := range rows {
		quietByParity[index%2] += boolToIndex(isQuietHihat(row[channelHihat]))
	}
	parity := boolToIndex(quietByParity[1] >= quietByParity[0])
	for index := range rows {
		volumes := [2]int{shakerSoftVolume, shakerAccentVolume}
		cell := TrackerCell{Note: note("C-9"), Instrument: instrumentShaker, Effect: effectVolume, Parameter: volumes[boolToIndex(index%4 >= 2)]}
		setRowCellIf(rows, index, channelShaker, cell, index%2 == parity && isQuietHihat(rows[index][channelHihat]))
	}
}

func isQuietHihat(hihat TrackerCell) bool {
	isSoft := hihat.Effect == effectVolume && hihat.Parameter < hihatLoudVolume
	return hihat.Note == noteEmpty || isSoft
}

func arrangeClap(rows [][trackerChannels]TrackerCell, _ []ChordTones, _ *Song) {
	snareRows := 0
	for _, row := range rows {
		snareRows += boolToIndex(isSnareCell(row[channelDrums]))
	}
	hasSnare := snareRows > 0
	for index := range rows {
		isBackbeat := index%backbeatSpacing == backbeatOffset
		isClap := isSnareCell(rows[index][channelDrums]) || (!hasSnare && isBackbeat)
		setRowCellIf(rows, index, channelClap, TrackerCell{Note: note("C-6"), Instrument: instrumentClap}, isClap)
	}
}

func isSnareCell(cell TrackerCell) bool {
	return cell.Note >= 0 && cell.Instrument == instrumentSnare
}

func arrangePad(rows [][trackerChannels]TrackerCell, chords []ChordTones, _ *Song) {
	for index := range rows {
		isChange := index == 0 || chords[index] != chords[index-1]
		chord := chords[index]
		cell := TrackerCell{Note: chord.Tone(0, padOctave), Instrument: instrumentPad, Effect: effectArpeggio, Parameter: chord.Third<<4 | chord.Fifth}
		setRowCellIf(rows, index, channelPad, cell, isChange)
	}
}

func arrangePluck(rows [][trackerChannels]TrackerCell, chords []ChordTones, song *Song) {
	spacing := 2 + 2*boolToIndex(song.Tempo < slowSongTempo)
	for index := 0; index < len(rows); index += spacing {
		step := pluckChordSteps[(index/spacing)%len(pluckChordSteps)] + 3*boolToIndex(index%8 == 4)
		rows[index][channelPluck] = TrackerCell{Note: chords[index].Tone(step, pluckOctave), Instrument: instrumentPluck}
	}
}

func arrangeOffbeat(rows [][trackerChannels]TrackerCell, chords []ChordTones, _ *Song) {
	for index := range rows {
		isOffbeat := index%4 == 2 && rows[index][channelDrums].Instrument != instrumentKick
		chord := chords[index]
		cell := TrackerCell{Note: chord.Tone(0, offbeatOctave), Instrument: instrumentOffbeat, Effect: effectArpeggio, Parameter: chord.Third<<4 | chord.Fifth}
		setRowCellIf(rows, index, channelOffbeat, cell, isOffbeat)
	}
}

func arrangeHarmony(rows [][trackerChannels]TrackerCell, chords []ChordTones, _ *Song) {
	for index := range rows {
		lead := rows[index][channelLead]
		cell := TrackerCell{Note: chordToneBelow(lead.Note, chords[index]), Instrument: instrumentHarmony}
		setRowCellIf(rows, index, channelHarmony, cell, lead.Note >= 0)
	}
}

func chordToneBelow(melodyNote int, chord ChordTones) int {
	best := melodyNote - harmonyLowestDrop
	for drop := 1; drop <= harmonyLowestDrop; drop++ {
		candidate := melodyNote - drop
		best = candidate + (best-candidate)*boolToIndex(best > candidate || !chord.Contains(candidate))
	}
	return best
}

func (c ChordTones) Contains(midiNote int) bool {
	interval := positiveModulo(midiNote-c.Root, 12)
	return interval == 0 || interval == positiveModulo(c.Third, 12) || interval == positiveModulo(c.Fifth, 12)
}

func setRowCellIf(rows [][trackerChannels]TrackerCell, row, channel int, cell TrackerCell, shouldSet bool) {
	if !shouldSet {
		return
	}
	rows[row][channel] = cell
}
