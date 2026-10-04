package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ===== Types =====

type TrackerCell struct {
	Note       int
	Instrument int
	Effect     byte
	Parameter  int
}

type Pattern struct {
	Rows [][trackerChannels]TrackerCell
}

type Song struct {
	Title       string
	Tempo       int
	Speed       int
	Instruments [maximumInstruments]VoiceSettings
	ChannelPans [trackerChannels]float32
	Layers      [trackerChannels]ChannelLayer
	TempoBoost  int
	Patterns    map[int]*Pattern
	Order       []int
}

type MusicSignal uint8

type ChannelLayer struct {
	Signal    MusicSignal
	Threshold float32
}

type ChannelState struct {
	Voice         Voice
	ArpeggioFirst int
	ArpeggioOther int
	Gain          float32
}

type Tracker struct {
	song             *Song
	channels         [trackerChannels]ChannelState
	orderIndex       int
	row              int
	tick             int
	samplesUntilTick float32
	seed             uint32
	signals          [musicSignalCount]float32
}

type songDirective func(song *Song, fields []string, parser *songParser) error

type songParser struct {
	currentPattern *Pattern
}

// ===== Constants =====

const (
	trackerChannels    = 10
	maximumInstruments = 32
	noteEmpty          = -1
	noteOff            = -2
	effectNone         = 0
	effectArpeggio     = '0'
	effectVolume       = 'C'
	maximumVolumeParam = 0x40
	layerFadeSeconds   = 0.9
)

const (
	SignalAlways MusicSignal = iota
	SignalHorde
	SignalBoss
	SignalReactions
	SignalDanger
	musicSignalCount
)

var musicSignalNames = map[string]MusicSignal{
	"always": SignalAlways, "horde": SignalHorde, "boss": SignalBoss, "reactions": SignalReactions, "danger": SignalDanger,
}

var noteSemitones = map[string]int{
	"C-": 0, "C#": 1, "D-": 2, "D#": 3, "E-": 4, "F-": 5, "F#": 6, "G-": 7, "G#": 8, "A-": 9, "A#": 10, "B-": 11,
}

var songDirectives = map[string]songDirective{
	"title":      parseTitle,
	"tempo":      parseTempo,
	"speed":      parseSpeed,
	"pan":        parsePan,
	"order":      parseOrder,
	"instrument": parseInstrument,
	"pattern":    parsePatternHeader,
	"layer":      parseLayer,
	"tempoboost": parseTempoBoost,
}

var instrumentProperties = map[string]func(settings *VoiceSettings, value float32){
	"attack":  func(settings *VoiceSettings, value float32) { settings.Envelope.Attack = value },
	"decay":   func(settings *VoiceSettings, value float32) { settings.Envelope.Decay = value },
	"sustain": func(settings *VoiceSettings, value float32) { settings.Envelope.Sustain = value },
	"release": func(settings *VoiceSettings, value float32) { settings.Envelope.Release = value },
	"volume":  func(settings *VoiceSettings, value float32) { settings.Volume = value },
	"duty":    func(settings *VoiceSettings, value float32) { settings.Duty = value },
	"slide":   func(settings *VoiceSettings, value float32) { settings.SlideOctaves = value },
	"filter":  func(settings *VoiceSettings, value float32) { settings.Filter = value },
	"sub":     func(settings *VoiceSettings, value float32) { settings.SubLevel = value },
	"detune":  func(settings *VoiceSettings, value float32) { settings.Detune = value },
	"drive":   func(settings *VoiceSettings, value float32) { settings.Drive = value },
}

// ===== Public API =====

func ParseSong(source string) (*Song, error) {
	song := &Song{Tempo: 125, Speed: 6, Patterns: map[int]*Pattern{}, ChannelPans: [trackerChannels]float32{0.3, 0.7, 0.7, 0.3, 0.4, 0.35, 0.65, 0.5, 0.7, 0.3}}
	parser := &songParser{}
	for lineNumber, line := range strings.Split(source, "\n") {
		if err := parser.parseLine(song, strings.TrimSpace(line)); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
	}
	return song, validateSong(song)
}

func NewTracker(song *Song) *Tracker {
	tracker := &Tracker{song: song, seed: 0x7AC4}
	tracker.signals[SignalAlways] = 1
	return tracker
}

func (t *Tracker) SetSignals(signals [musicSignalCount]float32) {
	t.signals = signals
	t.signals[SignalAlways] = 1
}

func (t *Tracker) Intensity() float32 {
	return max(t.signals[SignalHorde], min(1, t.signals[SignalBoss]), t.signals[SignalDanger])
}

func (t *Tracker) Render() (float32, float32) {
	t.samplesUntilTick--
	if t.samplesUntilTick <= 0 {
		t.processTick()
		t.samplesUntilTick += sampleRate * 2.5 / (float32(t.song.Tempo) + float32(t.song.TempoBoost)*t.Intensity())
	}
	left, right := float32(0), float32(0)
	for channel := range t.channels {
		channelLeft, channelRight := t.renderChannel(channel)
		left += channelLeft
		right += channelRight
	}
	return left, right
}

// ===== Internal =====

func (p *songParser) parseLine(song *Song, rawLine string) error {
	line := strings.TrimSpace(stripComment(rawLine))
	if line == "" {
		return nil
	}
	fields := strings.Fields(line)
	directive, isDirective := songDirectives[fields[0]]
	if isDirective {
		return directive(song, fields[1:], p)
	}
	return p.parseRow(song, line)
}

func stripComment(line string) string {
	if strings.HasPrefix(line, "#") {
		return ""
	}
	before, _, _ := strings.Cut(line, " #")
	return before
}

func (p *songParser) parseRow(song *Song, line string) error {
	if p.currentPattern == nil {
		return fmt.Errorf("row outside of a pattern: %q", line)
	}
	cells := strings.Split(line, "|")
	if len(cells) != trackerChannels {
		return fmt.Errorf("expected %d cells separated by |, got %d", trackerChannels, len(cells))
	}
	var row [trackerChannels]TrackerCell
	for channel, cell := range cells {
		parsed, err := parseCell(strings.Fields(cell))
		if err != nil {
			return fmt.Errorf("channel %d: %w", channel+1, err)
		}
		row[channel] = parsed
	}
	p.currentPattern.Rows = append(p.currentPattern.Rows, row)
	return nil
}

func parseCell(tokens []string) (TrackerCell, error) {
	cell := TrackerCell{Note: noteEmpty}
	if len(tokens) != 3 {
		return cell, fmt.Errorf("a cell needs note, instrument and effect, got %v", tokens)
	}
	note, err := parseNote(tokens[0])
	if err != nil {
		return cell, err
	}
	cell.Note = note
	cell.Instrument, err = parseOptionalNumber(tokens[1], 10)
	if err != nil {
		return cell, err
	}
	cell.Effect, cell.Parameter, err = parseEffect(tokens[2])
	return cell, err
}

func parseNote(token string) (int, error) {
	specialNotes := map[string]int{"---": noteEmpty, "===": noteOff}
	if note, isSpecial := specialNotes[token]; isSpecial {
		return note, nil
	}
	semitone, isKnown := noteSemitones[token[:min(2, len(token))]]
	octave, err := strconv.Atoi(token[min(2, len(token)):])
	if !isKnown || err != nil || len(token) != 3 {
		return noteEmpty, fmt.Errorf("invalid note %q (use C-4, C#4, --- or ===)", token)
	}
	return octave*12 + semitone, nil
}

func parseOptionalNumber(token string, base int) (int, error) {
	if strings.Trim(token, ".") == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(token, base, 32)
	return int(value), err
}

func parseEffect(token string) (byte, int, error) {
	if token == "..." {
		return effectNone, 0, nil
	}
	parameter, err := strconv.ParseInt(token[1:], 16, 32)
	if len(token) != 3 || err != nil {
		return effectNone, 0, fmt.Errorf("invalid effect %q (use 0xy, Cxx or ...)", token)
	}
	return token[0], int(parameter), nil
}

func parseTitle(song *Song, fields []string, _ *songParser) error {
	song.Title = strings.Join(fields, " ")
	return nil
}

func parseTempo(song *Song, fields []string, _ *songParser) error {
	return parseIntegerInto(&song.Tempo, fields)
}

func parseSpeed(song *Song, fields []string, _ *songParser) error {
	return parseIntegerInto(&song.Speed, fields)
}

func parseIntegerInto(target *int, fields []string) error {
	if len(fields) != 1 {
		return fmt.Errorf("expected one number")
	}
	value, err := strconv.Atoi(fields[0])
	*target = value
	return err
}

func parsePan(song *Song, fields []string, _ *songParser) error {
	if len(fields) != trackerChannels {
		return fmt.Errorf("pan needs %d values", trackerChannels)
	}
	for channel, field := range fields {
		value, err := strconv.ParseFloat(field, 32)
		if err != nil {
			return err
		}
		song.ChannelPans[channel] = float32(value)
	}
	return nil
}

func parseOrder(song *Song, fields []string, _ *songParser) error {
	for _, field := range fields {
		pattern, err := strconv.Atoi(field)
		if err != nil {
			return err
		}
		song.Order = append(song.Order, pattern)
	}
	return nil
}

func parseInstrument(song *Song, fields []string, _ *songParser) error {
	if len(fields) < 3 || len(fields)%2 != 1 {
		return fmt.Errorf("instrument needs: id name waveform then property value pairs")
	}
	id, err := strconv.Atoi(fields[0])
	waveform, isKnownWaveform := waveformNames[fields[2]]
	if err != nil || id < 1 || id >= maximumInstruments || !isKnownWaveform {
		return fmt.Errorf("invalid instrument id or waveform in %v", fields)
	}
	settings := VoiceSettings{Waveform: waveform, Volume: 1}
	for index := 3; index < len(fields); index += 2 {
		if err := applyInstrumentProperty(&settings, fields[index], fields[index+1]); err != nil {
			return err
		}
	}
	song.Instruments[id] = settings
	return nil
}

func applyInstrumentProperty(settings *VoiceSettings, name, rawValue string) error {
	setter, isKnown := instrumentProperties[name]
	value, err := strconv.ParseFloat(rawValue, 32)
	if !isKnown || err != nil {
		return fmt.Errorf("invalid instrument property %s %s", name, rawValue)
	}
	setter(settings, float32(value))
	return nil
}

func parsePatternHeader(song *Song, fields []string, parser *songParser) error {
	id, err := strconv.Atoi(strings.Join(fields, ""))
	if err != nil {
		return fmt.Errorf("pattern needs a numeric id")
	}
	parser.currentPattern = &Pattern{}
	song.Patterns[id] = parser.currentPattern
	return nil
}

func validateSong(song *Song) error {
	if len(song.Order) == 0 || song.Tempo <= 0 || song.Speed <= 0 {
		return fmt.Errorf("song needs an order list, a tempo and a speed")
	}
	for _, id := range song.Order {
		if pattern, exists := song.Patterns[id]; !exists || len(pattern.Rows) == 0 {
			return fmt.Errorf("order references missing or empty pattern %d", id)
		}
	}
	return nil
}

func (t *Tracker) processTick() {
	if t.tick == 0 {
		t.playRow()
	}
	t.applyArpeggio()
	t.tick++
	if t.tick < t.song.Speed {
		return
	}
	t.tick = 0
	t.row++
	pattern := t.song.Patterns[t.song.Order[t.orderIndex]]
	if t.row < len(pattern.Rows) {
		return
	}
	t.row = 0
	t.orderIndex = (t.orderIndex + 1) % len(t.song.Order)
}

func (t *Tracker) playRow() {
	pattern := t.song.Patterns[t.song.Order[t.orderIndex]]
	for channel, cell := range pattern.Rows[t.row] {
		t.playCell(channel, cell)
	}
}

func (t *Tracker) playCell(channel int, cell TrackerCell) {
	state := &t.channels[channel]
	state.updateArpeggio(cell)
	t.releaseIf(state, cell.Note == noteOff)
	if cell.Note < 0 {
		return
	}
	settings := t.song.Instruments[max(1, cell.Instrument)]
	settings.Frequency = NoteFrequency(cell.Note)
	settings.Pan = t.song.ChannelPans[channel]
	settings.Volume *= cellVolume(cell)
	t.seed++
	state.Voice.Start(settings, t.seed*2654435761)
}

func (state *ChannelState) updateArpeggio(cell TrackerCell) {
	isChanging := cell.Note >= 0 || cell.Effect != effectNone
	if !isChanging {
		return
	}
	isArpeggio := cell.Effect == effectArpeggio
	state.ArpeggioFirst = (cell.Parameter >> 4) * boolToIndex(isArpeggio)
	state.ArpeggioOther = (cell.Parameter & 0xF) * boolToIndex(isArpeggio)
}

func cellVolume(cell TrackerCell) float32 {
	isVolume := cell.Effect == effectVolume
	return 1 + (float32(min(cell.Parameter, maximumVolumeParam))/maximumVolumeParam-1)*boolToFloat(isVolume)
}

func (t *Tracker) releaseIf(state *ChannelState, shouldRelease bool) {
	if !shouldRelease {
		return
	}
	state.Voice.Release()
}

func (t *Tracker) applyArpeggio() {
	for channel := range t.channels {
		state := &t.channels[channel]
		offsets := [3]int{0, state.ArpeggioFirst, state.ArpeggioOther}
		state.Voice.FrequencyMul = float32(math.Exp2(float64(offsets[t.tick%3]) / 12))
	}
}

func (t *Tracker) renderChannel(channel int) (float32, float32) {
	state := &t.channels[channel]
	layer := &t.song.Layers[channel]
	target := boolToFloat(t.signals[layer.Signal] >= layer.Threshold)
	state.Gain += (target - state.Gain) / (layerFadeSeconds * sampleRate)
	if !state.Voice.IsActive {
		return 0, 0
	}
	left, right := state.Voice.Render()
	return left * state.Gain, right * state.Gain
}

func parseLayer(song *Song, fields []string, _ *songParser) error {
	if len(fields) != 3 {
		return fmt.Errorf("layer needs: channel signal threshold")
	}
	channel, channelErr := strconv.Atoi(fields[0])
	signal, isKnownSignal := musicSignalNames[fields[1]]
	threshold, thresholdErr := strconv.ParseFloat(fields[2], 32)
	if channelErr != nil || channel < 1 || channel > trackerChannels || !isKnownSignal || thresholdErr != nil {
		return fmt.Errorf("invalid layer %v (signals: always, horde, boss, reactions, danger)", fields)
	}
	song.Layers[channel-1] = ChannelLayer{Signal: signal, Threshold: float32(threshold)}
	return nil
}

func parseTempoBoost(song *Song, fields []string, _ *songParser) error {
	return parseIntegerInto(&song.TempoBoost, fields)
}
