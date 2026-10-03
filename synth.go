package main

import "math"

// ===== Types =====

type Waveform uint8

type Envelope struct {
	Attack  float32
	Decay   float32
	Sustain float32
	Release float32
}

type VoiceSettings struct {
	Waveform     Waveform
	Frequency    float32
	SlideOctaves float32
	Duty         float32
	Volume       float32
	Pan          float32
	Filter       float32
	Envelope     Envelope
	Duration     float32
	Delay        float32
	SubLevel     float32
	Detune       float32
}

type Voice struct {
	IsActive     bool
	Settings     VoiceSettings
	Frequency    float32
	FrequencyMul float32
	SlideFactor  float32
	Age          float32
	ReleaseAge   float32
	ReleaseLevel float32
	IsReleased   bool
	Phase        float32
	DetunedPhase float32
	SubPhase     float32
	DetuneRatio  float32
	NoiseValue   float32
	FilterState  float32
	random       Random
}

type waveformSampler func(phase, duty, noise float32) float32

// ===== Constants =====

const (
	WaveSquare Waveform = iota
	WaveSaw
	WaveTriangle
	WaveSine
	WaveNoise
	waveformCount
)

const (
	sampleRate          = 44100
	sampleSeconds       = 1.0 / sampleRate
	minimumEnvelopeTime = 0.0005
)

var waveformSamplers = [waveformCount]waveformSampler{
	WaveSquare:   func(phase, duty, _ float32) float32 { return 1 - 2*boolToFloat(phase >= duty) },
	WaveSaw:      func(phase, _, _ float32) float32 { return 2*phase - 1 },
	WaveTriangle: func(phase, _, _ float32) float32 { return 4*abs(phase-0.5) - 1 },
	WaveSine:     func(phase, _, _ float32) float32 { return sineOfPhase(phase) },
	WaveNoise:    func(_, _, noise float32) float32 { return noise },
}

var waveformNames = map[string]Waveform{
	"square": WaveSquare, "saw": WaveSaw, "triangle": WaveTriangle, "sine": WaveSine, "noise": WaveNoise,
}

// ===== Public API =====

func (v *Voice) Start(settings VoiceSettings, seed uint32) {
	*v = Voice{
		IsActive:     true,
		Settings:     settings,
		Frequency:    settings.Frequency,
		FrequencyMul: 1,
		SlideFactor:  float32(math.Exp2(float64(settings.SlideOctaves) * sampleSeconds)),
		DetuneRatio:  float32(math.Exp2(float64(settings.Detune) / 12)),
		random:       NewRandom(seed),
	}
	v.Settings.Duty = settings.Duty + 0.5*boolToFloat(settings.Duty == 0)
}

func (v *Voice) Release() {
	if v.IsReleased {
		return
	}
	v.ReleaseLevel = v.heldLevel()
	v.ReleaseAge = v.Age
	v.IsReleased = true
}

func (v *Voice) Render() (float32, float32) {
	if v.Settings.Delay > 0 {
		v.Settings.Delay -= sampleSeconds
		return 0, 0
	}
	amplitude := v.envelopeLevel()
	v.advancePhase()
	raw := v.oscillate()
	v.FilterState += (raw - v.FilterState) * (1 - v.Settings.Filter)
	value := v.FilterState * amplitude * v.Settings.Volume
	v.Age += sampleSeconds
	v.Frequency *= v.SlideFactor
	return value * (1 - v.Settings.Pan), value * v.Settings.Pan
}

func NoteFrequency(note int) float32 {
	return float32(440 * math.Exp2(float64(note-57)/12))
}

// ===== Internal =====

func (v *Voice) oscillate() float32 {
	sampler := waveformSamplers[v.Settings.Waveform]
	detuneMix := boolToFloat(v.Settings.Detune != 0)
	main := sampler(v.Phase, v.Settings.Duty, v.NoiseValue)
	detuned := sampler(v.DetunedPhase, v.Settings.Duty, v.NoiseValue) * detuneMix
	sub := sineOfPhase(v.SubPhase) * v.Settings.SubLevel
	return (main+detuned)/(1+detuneMix) + sub
}

func sineOfPhase(phase float32) float32 {
	return float32(math.Sin(2 * math.Pi * float64(phase)))
}

func wrapPhase(phase float32) float32 {
	return phase - float32(int(phase))
}

func (v *Voice) advancePhase() {
	step := v.Frequency * v.FrequencyMul * sampleSeconds
	v.Phase += step
	v.DetunedPhase = wrapPhase(v.DetunedPhase + step*v.DetuneRatio)
	v.SubPhase = wrapPhase(v.SubPhase + step/2)
	hasWrapped := v.Phase >= 1
	v.Phase = wrapPhase(v.Phase)
	freshNoise := v.random.Float()*2 - 1
	v.NoiseValue += (freshNoise - v.NoiseValue) * boolToFloat(hasWrapped)
}

func (v *Voice) heldLevel() float32 {
	envelope := &v.Settings.Envelope
	attackLevel := min(1, v.Age/max(envelope.Attack, minimumEnvelopeTime))
	decayProgress := clamp((v.Age-envelope.Attack)/max(envelope.Decay, minimumEnvelopeTime), 0, 1)
	return attackLevel * (1 - decayProgress*(1-envelope.Sustain))
}

func (v *Voice) envelopeLevel() float32 {
	envelope := &v.Settings.Envelope
	isExpired := v.Settings.Duration > 0 && v.Age >= v.Settings.Duration
	isFaded := envelope.Sustain == 0 && v.Age >= envelope.Attack+envelope.Decay
	if isExpired || isFaded {
		v.Release()
	}
	if !v.IsReleased {
		return v.heldLevel()
	}
	releaseProgress := (v.Age - v.ReleaseAge) / max(envelope.Release, minimumEnvelopeTime)
	v.IsActive = releaseProgress < 1
	return v.ReleaseLevel * max(0, 1-releaseProgress)
}
