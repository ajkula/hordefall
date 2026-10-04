package audio

// ===== Types =====

type SoundKind uint8

type SoundLayer struct {
	Settings    VoiceSettings
	PitchJitter float32
}

type SoundDefinition struct {
	Layers          []SoundLayer
	MinimumInterval float32
}

// ===== Constants =====

const LaserChargeSoundSeconds = 2.6

const (
	SoundNone SoundKind = iota
	SoundPop
	SoundThud
	SoundPew
	SoundZap
	SoundNova
	SoundSplash
	SoundRain
	SoundExplosion
	SoundShatter
	SoundElectrocute
	SoundFreeze
	SoundHiss
	SoundHurt
	SoundDash
	SoundLevelUp
	SoundGem
	SoundStomp
	SoundLaserCharge
	SoundLaserFire
	SoundMenuMove
	SoundMenuSelect
	soundKindCount
)

var soundTable = [soundKindCount]SoundDefinition{
	SoundPop: {MinimumInterval: 0.022, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveTriangle, Frequency: 900, SlideOctaves: -5, Volume: 0.32, Envelope: Envelope{Attack: 0.001, Decay: 0.07}}, PitchJitter: 0.25},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 7000, Volume: 0.1, Filter: 0.3, Envelope: Envelope{Attack: 0.001, Decay: 0.025}}, PitchJitter: 0.2},
	}},
	SoundThud: {MinimumInterval: 0.05, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSine, Frequency: 160, SlideOctaves: -3, Volume: 0.5, Envelope: Envelope{Attack: 0.002, Decay: 0.16}}, PitchJitter: 0.15},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 900, Volume: 0.2, Filter: 0.6, Envelope: Envelope{Attack: 0.001, Decay: 0.1}}},
	}},
	SoundPew: {MinimumInterval: 0.04, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 1500, SlideOctaves: -6, Duty: 0.25, Volume: 0.1, Envelope: Envelope{Attack: 0.001, Decay: 0.06}}, PitchJitter: 0.1},
	}},
	SoundZap: {MinimumInterval: 0.06, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSaw, Frequency: 220, SlideOctaves: 2, Volume: 0.18, Envelope: Envelope{Attack: 0.001, Decay: 0.12}}, PitchJitter: 0.3},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 3000, Volume: 0.12, Filter: 0.2, Envelope: Envelope{Attack: 0.001, Decay: 0.1}}},
	}},
	SoundNova: {MinimumInterval: 0.1, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveTriangle, Frequency: 1600, SlideOctaves: -3, Volume: 0.2, Envelope: Envelope{Attack: 0.005, Decay: 0.25}}},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 4000, Volume: 0.1, Filter: 0.7, Envelope: Envelope{Attack: 0.01, Decay: 0.3}}},
	}},
	SoundSplash: {MinimumInterval: 0.05, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 1800, Volume: 0.25, Filter: 0.75, Envelope: Envelope{Attack: 0.002, Decay: 0.18}}, PitchJitter: 0.2},
		{Settings: VoiceSettings{Waveform: WaveSine, Frequency: 300, SlideOctaves: -2, Volume: 0.15, Envelope: Envelope{Attack: 0.002, Decay: 0.12}}},
	}},
	SoundRain: {MinimumInterval: 0.1, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 5000, Volume: 0.14, Filter: 0.5, Envelope: Envelope{Attack: 0.02, Decay: 0.5}}},
	}},
	SoundExplosion: {MinimumInterval: 0.06, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 1200, Volume: 0.5, Filter: 0.55, Envelope: Envelope{Attack: 0.002, Decay: 0.45}}, PitchJitter: 0.2},
		{Settings: VoiceSettings{Waveform: WaveSine, Frequency: 110, SlideOctaves: -2.5, Volume: 0.55, Envelope: Envelope{Attack: 0.002, Decay: 0.35}}, PitchJitter: 0.1},
	}},
	SoundShatter: {MinimumInterval: 0.05, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 2600, SlideOctaves: -1, Duty: 0.2, Volume: 0.12, Envelope: Envelope{Attack: 0.001, Decay: 0.12}}, PitchJitter: 0.3},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 9000, Volume: 0.16, Filter: 0.1, Envelope: Envelope{Attack: 0.001, Decay: 0.08}}},
	}},
	SoundElectrocute: {MinimumInterval: 0.07, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSaw, Frequency: 90, Volume: 0.22, Envelope: Envelope{Attack: 0.002, Decay: 0.25}}, PitchJitter: 0.2},
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 180, Duty: 0.1, Volume: 0.12, Envelope: Envelope{Attack: 0.002, Decay: 0.2}}, PitchJitter: 0.2},
	}},
	SoundFreeze: {MinimumInterval: 0.05, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveTriangle, Frequency: 2400, SlideOctaves: 1.5, Volume: 0.12, Envelope: Envelope{Attack: 0.002, Decay: 0.15}}, PitchJitter: 0.2},
	}},
	SoundHiss: {MinimumInterval: 0.06, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 6000, Volume: 0.14, Filter: 0.4, Envelope: Envelope{Attack: 0.01, Decay: 0.25}}},
	}},
	SoundHurt: {MinimumInterval: 0.1, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 220, SlideOctaves: -3, Volume: 0.28, Envelope: Envelope{Attack: 0.002, Decay: 0.18}}},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 800, Volume: 0.2, Filter: 0.6, Envelope: Envelope{Attack: 0.002, Decay: 0.1}}},
	}},
	SoundDash: {MinimumInterval: 0.1, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 3000, SlideOctaves: 2, Volume: 0.16, Filter: 0.5, Envelope: Envelope{Attack: 0.005, Decay: 0.12}}},
		{Settings: VoiceSettings{Waveform: WaveSine, Frequency: 300, SlideOctaves: 4, Volume: 0.15, Envelope: Envelope{Attack: 0.005, Decay: 0.1}}},
	}},
	SoundLevelUp: {MinimumInterval: 0.2, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveTriangle, Frequency: 523, Volume: 0.25, Envelope: Envelope{Attack: 0.002, Decay: 0.15}}},
		{Settings: VoiceSettings{Waveform: WaveTriangle, Frequency: 659, Volume: 0.25, Delay: 0.08, Envelope: Envelope{Attack: 0.002, Decay: 0.15}}},
		{Settings: VoiceSettings{Waveform: WaveTriangle, Frequency: 784, Volume: 0.25, Delay: 0.16, Envelope: Envelope{Attack: 0.002, Decay: 0.15}}},
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 1046, Duty: 0.25, Volume: 0.18, Delay: 0.24, Envelope: Envelope{Attack: 0.002, Decay: 0.35}}},
	}},
	SoundGem: {MinimumInterval: 0.03, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSine, Frequency: 1300, SlideOctaves: 2, Volume: 0.07, Envelope: Envelope{Attack: 0.001, Decay: 0.04}}, PitchJitter: 0.15},
	}},
	SoundStomp: {MinimumInterval: 0.04, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSine, Frequency: 90, SlideOctaves: -2.5, Volume: 0.55, Envelope: Envelope{Attack: 0.002, Decay: 0.22}}, PitchJitter: 0.1},
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 600, Volume: 0.25, Filter: 0.7, Envelope: Envelope{Attack: 0.002, Decay: 0.15}}},
	}},
	SoundLaserCharge: {MinimumInterval: 0.2, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSaw, Frequency: 110, SlideOctaves: 1.3, Volume: 0.16, Filter: 0.6, Duration: LaserChargeSoundSeconds, Envelope: Envelope{Attack: 0.4, Decay: 0.1, Sustain: 1, Release: 0.08}}},
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 220, SlideOctaves: 1.3, Duty: 0.1, Volume: 0.06, Duration: LaserChargeSoundSeconds, Envelope: Envelope{Attack: 0.4, Decay: 0.1, Sustain: 1, Release: 0.08}}},
	}},
	SoundLaserFire: {MinimumInterval: 0.2, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveNoise, Frequency: 2500, Volume: 0.55, Filter: 0.35, Envelope: Envelope{Attack: 0.005, Decay: 0.5}}},
		{Settings: VoiceSettings{Waveform: WaveSaw, Frequency: 70, SlideOctaves: -0.5, Volume: 0.45, Envelope: Envelope{Attack: 0.005, Decay: 0.45}}},
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 140, Volume: 0.2, Envelope: Envelope{Attack: 0.005, Decay: 0.3}}},
	}},
	SoundMenuMove: {MinimumInterval: 0.03, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 900, Duty: 0.25, Volume: 0.1, Envelope: Envelope{Attack: 0.001, Decay: 0.04}}},
	}},
	SoundMenuSelect: {MinimumInterval: 0.05, Layers: []SoundLayer{
		{Settings: VoiceSettings{Waveform: WaveSquare, Frequency: 600, SlideOctaves: 3, Duty: 0.25, Volume: 0.14, Envelope: Envelope{Attack: 0.001, Decay: 0.1}}},
	}},
}
