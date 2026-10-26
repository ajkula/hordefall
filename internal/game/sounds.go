package game

import "github.com/ajkula/ajkurig/audio"

// ===== Types =====

type SoundKind uint8

type SoundIDs [soundKindCount]audio.SoundID

// ===== Constants =====

const LaserChargeSoundSeconds = 2.6

const FrostChargeSoundSeconds = 4.4

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
	SoundBombWhistle
	SoundBigImpact
	SoundGlassShatter
	SoundChatter
	SoundWormRumble
	SoundWormRoar
	SoundFrostCharge
	SoundFrostFire
	SoundWormRoll
	soundKindCount
)

var frameworkSounds = map[SoundKind]audio.SoundID{
	SoundMenuMove:     audio.SoundMenuMove,
	SoundMenuSelect:   audio.SoundMenuSelect,
	SoundChatter:      audio.SoundVoiceChirp,
	SoundBombWhistle:  audio.SoundBombWhistle,
	SoundBigImpact:    audio.SoundBigImpact,
	SoundGlassShatter: audio.SoundGlassShatter,
}

var soundTable = map[SoundKind]audio.SoundDefinition{
	SoundPop: {Name: "pop", MinimumInterval: 0.022, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 900, SlideOctaves: -5, Volume: 0.32, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.07}}, PitchJitter: 0.25},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 7000, Volume: 0.1, Filter: 0.3, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.025}}, PitchJitter: 0.2},
	}},
	SoundThud: {Name: "thud", MinimumInterval: 0.05, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 160, SlideOctaves: -3, Volume: 0.5, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.16}}, PitchJitter: 0.15},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 900, Volume: 0.2, Filter: 0.6, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.1}}},
	}},
	SoundPew: {Name: "pew", MinimumInterval: 0.04, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 1500, SlideOctaves: -6, Duty: 0.25, Volume: 0.1, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.06}}, PitchJitter: 0.1},
	}},
	SoundZap: {Name: "zap", MinimumInterval: 0.06, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSaw, Frequency: 220, SlideOctaves: 2, Volume: 0.18, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.12}}, PitchJitter: 0.3},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 3000, Volume: 0.12, Filter: 0.2, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.1}}},
	}},
	SoundNova: {Name: "nova", MinimumInterval: 0.1, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 1600, SlideOctaves: -3, Volume: 0.2, Envelope: audio.Envelope{Attack: 0.005, Decay: 0.25}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 4000, Volume: 0.1, Filter: 0.7, Envelope: audio.Envelope{Attack: 0.01, Decay: 0.3}}},
	}},
	SoundSplash: {Name: "splash", MinimumInterval: 0.05, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 1800, Volume: 0.25, Filter: 0.75, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.18}}, PitchJitter: 0.2},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 300, SlideOctaves: -2, Volume: 0.15, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.12}}},
	}},
	SoundRain: {Name: "rain", MinimumInterval: 0.1, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 5000, Volume: 0.14, Filter: 0.5, Envelope: audio.Envelope{Attack: 0.02, Decay: 0.5}}},
	}},
	SoundExplosion: {Name: "explosion", MinimumInterval: 0.06, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 1200, Volume: 0.5, Filter: 0.55, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.45}}, PitchJitter: 0.2},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 110, SlideOctaves: -2.5, Volume: 0.55, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.35}}, PitchJitter: 0.1},
	}},
	SoundShatter: {Name: "shatter", MinimumInterval: 0.05, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 2600, SlideOctaves: -1, Duty: 0.2, Volume: 0.12, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.12}}, PitchJitter: 0.3},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 9000, Volume: 0.16, Filter: 0.1, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.08}}},
	}},
	SoundElectrocute: {Name: "electrocute", MinimumInterval: 0.07, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSaw, Frequency: 90, Volume: 0.22, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.25}}, PitchJitter: 0.2},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 180, Duty: 0.1, Volume: 0.12, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.2}}, PitchJitter: 0.2},
	}},
	SoundFreeze: {Name: "freeze", MinimumInterval: 0.05, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 2400, SlideOctaves: 1.5, Volume: 0.12, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.15}}, PitchJitter: 0.2},
	}},
	SoundHiss: {Name: "hiss", MinimumInterval: 0.06, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 6000, Volume: 0.14, Filter: 0.4, Envelope: audio.Envelope{Attack: 0.01, Decay: 0.25}}},
	}},
	SoundHurt: {Name: "hurt", MinimumInterval: 0.1, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 220, SlideOctaves: -3, Volume: 0.28, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.18}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 800, Volume: 0.2, Filter: 0.6, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.1}}},
	}},
	SoundDash: {Name: "dash", MinimumInterval: 0.1, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 3000, SlideOctaves: 2, Volume: 0.16, Filter: 0.5, Envelope: audio.Envelope{Attack: 0.005, Decay: 0.12}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 300, SlideOctaves: 4, Volume: 0.15, Envelope: audio.Envelope{Attack: 0.005, Decay: 0.1}}},
	}},
	SoundLevelUp: {Name: "level_up", MinimumInterval: 0.2, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 523, Volume: 0.25, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.15}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 659, Volume: 0.25, Delay: 0.08, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.15}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 784, Volume: 0.25, Delay: 0.16, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.15}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 1046, Duty: 0.25, Volume: 0.18, Delay: 0.24, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.35}}},
	}},
	SoundGem: {Name: "gem", MinimumInterval: 0.03, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 1300, SlideOctaves: 2, Volume: 0.07, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.04}}, PitchJitter: 0.15},
	}},
	SoundStomp: {Name: "stomp", MinimumInterval: 0.04, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 90, SlideOctaves: -2.5, Volume: 0.55, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.22}}, PitchJitter: 0.1},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 600, Volume: 0.25, Filter: 0.7, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.15}}},
	}},
	SoundLaserCharge: {Name: "laser_charge", MinimumInterval: 0.2, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSaw, Frequency: 110, SlideOctaves: 1.3, Volume: 0.16, Filter: 0.6, Duration: LaserChargeSoundSeconds, Envelope: audio.Envelope{Attack: 0.4, Decay: 0.1, Sustain: 1, Release: 0.08}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 220, SlideOctaves: 1.3, Duty: 0.1, Volume: 0.06, Duration: LaserChargeSoundSeconds, Envelope: audio.Envelope{Attack: 0.4, Decay: 0.1, Sustain: 1, Release: 0.08}}},
	}},
	SoundLaserFire: {Name: "laser_fire", MinimumInterval: 0.2, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 2500, Volume: 0.55, Filter: 0.35, Envelope: audio.Envelope{Attack: 0.005, Decay: 0.5}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSaw, Frequency: 70, SlideOctaves: -0.5, Volume: 0.45, Envelope: audio.Envelope{Attack: 0.005, Decay: 0.45}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 140, Volume: 0.2, Envelope: audio.Envelope{Attack: 0.005, Decay: 0.3}}},
	}},
	SoundWormRumble: {Name: "worm_rumble", MinimumInterval: 0.5, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 300, Volume: 0.5, Filter: 0.93, SubLevel: 0.5, Envelope: audio.Envelope{Attack: 0.08, Decay: 1}}, PitchJitter: 0.1},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 48, SlideOctaves: -0.5, Volume: 0.5, Drive: 1.5, Envelope: audio.Envelope{Attack: 0.05, Decay: 1.1}}},
	}},
	SoundWormRoar: {Name: "worm_roar", MinimumInterval: 0.5, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSaw, Frequency: 140, SlideOctaves: -1.6, Volume: 0.35, Filter: 0.6, Drive: 3, SubLevel: 0.6, Unison: 3, Detune: 0.3, Envelope: audio.Envelope{Attack: 0.03, Decay: 0.9}}, PitchJitter: 0.12},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 700, Volume: 0.35, Filter: 0.8, Envelope: audio.Envelope{Attack: 0.01, Decay: 0.6}}},
	}},
	SoundFrostCharge: {Name: "frost_charge", MinimumInterval: 0.5, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveTriangle, Frequency: 330, SlideOctaves: 0.45, Volume: 0.12, Unison: 5, Detune: 0.25, Duration: FrostChargeSoundSeconds, Envelope: audio.Envelope{Attack: 1.2, Decay: 0.1, Sustain: 1, Release: 0.1}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 1320, SlideOctaves: 0.45, Volume: 0.05, Duration: FrostChargeSoundSeconds, Envelope: audio.Envelope{Attack: 1.5, Decay: 0.1, Sustain: 1, Release: 0.1}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 5000, Volume: 0.05, Filter: 0.2, Duration: FrostChargeSoundSeconds, Envelope: audio.Envelope{Attack: 2, Decay: 0.1, Sustain: 1, Release: 0.1}}},
	}},
	SoundFrostFire: {Name: "frost_fire", MinimumInterval: 0.2, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 6000, Volume: 0.5, Filter: 0.15, Envelope: audio.Envelope{Attack: 0.003, Decay: 0.7}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSine, Frequency: 70, SlideOctaves: -1.5, Volume: 0.6, Drive: 1.5, Envelope: audio.Envelope{Attack: 0.002, Decay: 0.5}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSquare, Frequency: 2600, SlideOctaves: -2, Duty: 0.2, Volume: 0.1, Crush: 0.4, Envelope: audio.Envelope{Attack: 0.001, Decay: 0.4}}},
	}},
	SoundWormRoll: {Name: "worm_roll", MinimumInterval: 0.5, Layers: []audio.SoundLayer{
		{Settings: audio.VoiceSettings{Waveform: audio.WaveSaw, Frequency: 55, SlideOctaves: 0.8, Volume: 0.35, Filter: 0.7, Drive: 2, SubLevel: 0.5, Envelope: audio.Envelope{Attack: 0.15, Decay: 0.9}}},
		{Settings: audio.VoiceSettings{Waveform: audio.WaveNoise, Frequency: 400, Volume: 0.3, Filter: 0.88, Envelope: audio.Envelope{Attack: 0.1, Decay: 0.9}}},
	}},
}

// ===== Public API =====

func RegisterSounds(bank *audio.SoundBank) SoundIDs {
	var ids SoundIDs
	for kind := SoundNone + 1; kind < soundKindCount; kind++ {
		ids[kind] = registerSound(bank, kind)
	}
	return ids
}

// ===== Internal =====

func registerSound(bank *audio.SoundBank, kind SoundKind) audio.SoundID {
	if id, isShared := frameworkSounds[kind]; isShared {
		return id
	}
	return bank.Add(soundTable[kind])
}
