# Hordefall

A systemic horde survivor written in Go with [Ebitengine](https://ebitengine.org).
Thousands of enemies, auto-firing elemental weapons, and a living ground: every element interacts with enemies, with the terrain, and with each other.

## Run

```sh
go run .
```

## Menus, score and high score

- **Main menu**: Play, Options, Quit, and your high score (dark orange).
- **Options**: Configure buttons, Graphics (Fullscreen, also F11 anywhere; VSync; Resolution 720p / 1080p / Native, where Native renders at the screen's real pixel count for sharp lines and text; Show FPS; Screen shake Off / 50% / 100%; Effects Low / Medium / High, which keeps 30% / 60% / 100% of particles; Bloom Off / Soft / Strong, a glow on the world only (fire, lightning, explosions), never on the HUD; CRT filter, with curved screen, scanlines, RGB mask, colour fringing and vignette), Music on/off (sound effects stay on), Playlist (switch each track on or off: enabling a track plays it at once, disabling the current one moves to the next enabled track, rotation goes to the next enabled track; all off means no music), Back. Settings are saved to `%AppData%\hordefall\settings.json` and restored on launch; the playlist is a `"tracks": { "title": true/false }` map, and a track missing from it counts as enabled. Behind it runs an attract-mode benchmark: six 5-second gameplay sequences on loop (Inferno chain, Electrocution, Freeze and shatter, Wildfire, Spider tank, Horde x4000) interleaved with randomly generated LIVE ACTION clips (random weapons and levels, passives, enemy mix and count, starting statuses, prepared ground, sometimes a spider tank), all played by an autopilot Tachikoma, with live FPS, TPS, simulation time and entity counts.
- **Score**: every kill is worth its experience x 10 (a spider tank is worth 1,500) and every second survived is worth 5. Score and high score are shown in the HUD.
- **High score** is saved to `%AppData%\hordefall\highscore.json` on game over, when leaving a run for the main menu, on Quit and when the window is closed.
- **Pause** (Start / Esc) freezes everything, including screen shake and fire flicker: Resume, Music on/off, Back to main menu.
- **Game over**: Start / Enter to try again, Select / Backspace for the main menu.
- **Level up**: when several upgrades are available the game pauses on a choice of cards; when only one is possible it is applied at once, with a LEVEL UP! banner and no interruption. Once every weapon is maxed and only passives remain, each level up applies the next remaining passive in turn (a b c d a b c d, then a b d a b d once c is maxed), also without pausing.

## Music and sound

### Adaptive tracker music
The soundtrack is a small Amiga-style tracker running live: `music/theme.trk` is a plain-text song with 10 channels, patterns of rows, an order list and synthesized instruments (square with duty cycle, saw, triangle, sine, noise; ADSR envelope, pitch slide, low-pass filter). Effects: `0xy` arpeggio (the classic Amiga chord shimmer) and `Cxx` volume.

The music is procedural in the sense that it follows the game. Every channel always plays in time, but is only heard while its game signal is above a threshold, with a smooth fade:

| Layer | Signal | Meaning |
|-------|--------|---------|
| drums, bass | `always` | the base groove |
| hi-hats, chords, lead | `horde` | enemies massing around you |
| boss voices | `boss` | one deep voice per spider tank alive: saw stabs on the root (1 tank), a lower square growl on the fifth (2 tanks), the deepest saw drone sliding a semitone up and a tone down (3 tanks) |
| crystal bells | `reactions` | infernos, freezes, shatters chaining |
| tension pulse | `danger` | low health or a spider beam charging |

The tempo also rises with intensity (`tempoboost`). M mutes the music.

### Composing
Edit `music/theme.trk` and rebuild: it is embedded in the executable. A row is 10 cells separated by `|`, each cell is `NOTE INSTRUMENT EFFECT`, for example `A-4 05 037` (A4, instrument 5, minor-chord arpeggio). `---` means nothing, `===` releases the note, `..` and `...` leave instrument and effect empty. Instruments, layers (`layer <channel> <signal> <threshold>`), tempo and pan are declared at the top of the file. Parse errors give the line and channel, and `go test -run Theme` checks the song.

### Song rotation and procedural composer
The first song is picked at random among enabled tracks at launch. Then the next enabled track plays when a run starts, every 5 minutes during a run, and on the main menu after every full cycle of the attract-mode sequences, with a 3-second crossfade and a small banner: the hand-written theme first, then five songs composed procedurally at each launch from the styles in `composer.go`:

| Style | Mood | Key / mode | Tempo |
|-------|------|------------|-------|
| Neon Pursuit | fast, tense chase | E harmonic minor | 145 |
| Frozen Wastes | cold, airy, sparse | D dorian | 100 |
| Ember March | heavy half-time march | C phrygian | 112 |
| Skyline Rush | bright, heroic | F major | 155 |
| Grey Transmission | post-punk: driving melodic bass, overdriven chorused guitars | B minor | 160 |

A style sets the scale, tonic, chord progression, tempo, drum, bass, chord and melody rhythms, and timbres. The composer derives arpeggios from the scale chords and writes the melody as a random walk on the scale, anchored on chord tones on downbeats, so melodies differ at every launch. All songs share the same adaptive layers. Add a style by adding an entry to `musicStyles`.

Preview all songs as 30-second WAV files:

```powershell
$env:HORDEFALL_WAV_DIR = "./music/previews"; go test -run ExportSongPreviews
```

### Procedural sound effects
Every sound effect is synthesized on the fly from layered voices with a random pitch variation, so no two pops are identical: enemy pops, thuds and splashes, weapon pews, zaps and novas, reaction explosions, shatters, hisses and electrocutions, spider stomps, beam charge and fire, dash, hurt, level-up arpeggio, gem pickup and menu blips. Each sound has a minimum interval, so a hundred kills in a frame never saturate the mix. Sounds are declared in `sounds.go`; enemies, reactions and weapons reference them from their own tables.

## Controls

| Action        | Arcade stick / gamepad  | Keyboard        |
|---------------|-------------------------|-----------------|
| Move          | Stick (D-pad or analog) | WASD / arrows   |
| Aim           | Right stick (twin-stick pads) | follows movement |
| Fire (hold)   | Button 1                | J / Z           |
| Aim lock (hold) | Button 2              | K / X           |
| Dash          | Button 3                | Space / L / C   |
| Pause         | Start                   | Esc / P         |
| Configure buttons (title) | Select      | F2              |
| Input debug (in game) | Select          | F1              |

Aim follows your movement. Hold aim lock to freeze it and strafe or retreat while firing.
On a gamepad with two sticks, the right stick aims (twin-stick): move with the left stick, aim with the right one; pushing it past the deadzone also fires automatically, so no button is needed; release it and the aim follows your movement again. Fire still works on button 1, which is how the arcade stick plays.
Ember Bolt, Arc Lightning, Oil Flask and Downpour fire where you aim while Fire is held. Frost Nova and Orbit Blades are automatic.

On the title screen, Select / F2 opens the button setup: press your Fire, Aim lock and Dash buttons in turn.
The mapping is saved to `%AppData%\hordefall\controls.json` (raw button indices) and loaded on startup.

## The systems

**Statuses** (`status.go`): Burning, Chilled, Frozen, Oiled, Wet, Shocked. Each is a bit in `StatusFlags` with a duration, tint, speed factor and damage over time.

**Reactions**: an element hitting an enemy that carries a status triggers a reaction from `reactionRules`:

| Element  | Status           | Reaction     | Effect                                         |
|----------|------------------|--------------|------------------------------------------------|
| Fire     | Oiled            | INFERNO      | Fire explosion, chains, ignites the ground, hurts you |
| Shock    | Oiled            | INFERNO      | Sparks ignite oil                              |
| Oil      | Burning          | INFERNO      | Burning enemy walks into oil                   |
| Fire     | Chilled / Frozen | STEAM        | Scalding burst that wets around                |
| Frost    | Burning          | STEAM        |                                                |
| Frost    | Wet / Chilled    | FREEZE       | Enemy frozen solid                             |
| Shock    | Wet              | ELECTROCUTE  | Arcs to every wet enemy nearby                 |
| Physical / Shock | Frozen   | SHATTER      | x3 damage and ice shards                       |
| Fire / Water | Wet / Burning | HISS        | Extinguished                                   |

Reaction bursts carry an element too, so reactions chain into further reactions.

**Ground** (`ground.go`): a 256x256 cellular grid. Grass catches fire and spreads it, oil burns explosively, ice melts, water freezes under frost, ash regrows into grass. Standing on a cell applies its element to enemies (fire burns, water wets, oil soaks, ice chills) and fire hurts the player.

**Enemies** (`enemies.go`): Bloaters burst into oil, Frostlings leave ice and chill, Emberlings leave fire. The horde feeds the systems.

## The player: a Tachikoma

The player is a small blue Tachikoma seen from above, animated procedurally from the reference design: a front cabin with three eyes, a heavy abdomen on a ball joint, four white droplet-shaped legs ending in wheels, and two small manipulator arms.
- Each wheel follows its rest pose through a damped spring, so legs stretch, lag and overshoot when you accelerate or turn. Knees are solved with two-bone inverse kinematics.
- The abdomen hangs on a spring driven by acceleration: it trails, swings in turns and settles.
- The eyes track your aim; when you stand still for a while, they glance around with curiosity.
- The head is a turret on a neck bearing, driven by an angular servo (damped spring, slight overshoot) toward your aim, limited to about 110 degrees from the chassis. When the aim goes past that stop, the chassis pivots to follow; omni wheels let it face one way while rolling another. The abdomen counter-rotates a little as a counterweight.
- A barrel under the head recoils on every shot and projectiles leave from its muzzle. Arms follow the head.
- Legs tuck in during a dash. The body breathes, faster when rolling.
- Skid trails: sharp turns, braking and dashes leave a luminous white trail behind each wheel, as wide as the wheel, that drops quickly in intensity then lingers for about 1.6 s (`skids.go`). Purely visual, handling is unchanged.
Tuning: constants and `tachikomaLegLayouts` in `tachikoma.go`, colors and shapes in `tachikoma_render.go`.

## Boss: Spider Tank

Each boss event needs its own kills: the counter resets when a boss arrives, so kills never carry over to the next one. The requirement grows with the run: 300 kills at the start, 300 × (1 + minutes / 2) after that (1,050 at 5 minutes, 1,800 at 10, 3,300 at 20). A Ghost in the Shell style spider tank then walks in (at most 3 at once).
Its four legs are procedural: each leg is a two-bone chain solved with inverse kinematics (the knee is the outward solution of the two-circle intersection), each foot stays planted until it drifts too far from its rest pose, then steps ahead of the body with an eased arc. Legs move in diagonal pairs.
Every footfall is a stomp: dust, screen shake, a physical burst that crushes nearby enemies, shatters frozen ones and hurts you.
Its cockpit, sensor and cannon sit on a turret ring that tracks you with a slow, heavy servo, independently of where the legs carry the body.
**Devastator beam.** It spawns off screen and opens with its signature attack, so you know it is there:
1. Charging (2.2 s): the ground crackles with sparks and arcs along the line of fire, and a thin, intense targeting beam leaves the cannon and tracks you through the turret servo.
2. Locked (400 ms): the beam freezes and blinks. This is your window to get out, or to dash through.
3. Firing (0.35 s): a long, thick, blinding ray. It shreds your health (dashing makes you immune), burns every enemy it crosses, and scorches the ground: grass and oil catch fire, ice melts, dirt turns to ash.
4. Cooldown (6 to 9 s), then again.
Enemies killed by a boss give no experience, no kill and no lifesteal. Timings, damage and range are constants in `spider_laser.go`.

It takes statuses and reactions like any enemy, and explodes into a firestorm when destroyed.
Tuning lives in the constants and `spiderLegLayouts` of `spider.go`; drawing is in `spider_render.go`.

## Boss event: the Horde

After three spider tanks, the fourth boss event is a horde closing in from every side: 500 the first time, then 1,000, 1,500 and so on. Its core is a mix of brutes, frostlings, runners and swarmers, all twice as tough as usual; the explosive enemies (bloaters and emberlings, one in eight) form the outer shell, so the horde no longer goes up in one chain reaction. It counts as a boss: its purple bar shows how many are left, no other boss arrives until it is wiped out, and the music adds a boss voice. Horde kills give experience and score but do not advance the boss counter. Clearing it awards a bonus of 20 points per member. Tuning in `horde_event.go`.

## Weapons

Ember Bolt (fire), Frost Nova (frost, freezes water), Arc Lightning (shock chains), Oil Flask (soaks ground and enemies), Downpour (wets, douses fires), Orbit Blades (physical, shatters frozen). Up to 5 weapons, level 7 each, plus 6 passives.

## Architecture

| File             | Role                                                       |
|------------------|------------------------------------------------------------|
| `main.go`        | Game loop, state machine table                             |
| `status.go`      | Statuses, elements, reaction table                         |
| `combat.go`      | Hits, reactions, area bursts, deaths                       |
| `horde.go`       | Enemy movement, separation, contagion, spawn director      |
| `enemies.go`     | Enemy table and struct-of-arrays store                     |
| `weapons.go`     | Weapon table and behaviors                                 |
| `projectiles.go` | Bolts and lobbed flasks                                    |
| `ground.go`      | Living terrain cellular automaton                          |
| `upgrades.go`    | Passives and level-up offers                               |
| `player.go`      | Player movement, dash, damage, experience                  |
| `gems.go`        | Experience gems                                            |
| `effects.go`     | Particles, rings, lightning, popups, screen shake          |
| `spatial.go`     | Uniform grid with counting sort for neighbor queries       |
| `render.go`      | Batched circle sprites with `DrawTriangles32`, additive glow |
| `tachikoma.go`   | Player rig: wheel springs, abdomen sway, gaze, recoil      |
| `tachikoma_render.go` | Player drawing                                        |
| `skids.go`       | Wheel skid trails                                          |
| `horde_event.go` | Horde boss event: spawning, tracking, bar                  |
| `graphics.go`    | Graphics options, render scale, scaled text and rectangles |
| `postfx.go`      | Kage shaders: bloom (bright pass, blur) and CRT filter     |
| `spider.go`      | Spider tank boss: spawn, procedural legs, gait, stomps     |
| `spider_laser.go` | Spider tank beam: charge, lock, fire state machine         |
| `spider_render.go` | Spider tank drawing                                      |
| `ui.go`          | HUD, menus, overlays                                       |
| `input.go`       | Keyboard and gamepad bindings, raw button capture          |
| `bindings.go`    | Button mapping persistence                                 |
| `menus.go`       | Main menu, pause, game over and remap states               |
| `score.go`       | Score, high score persistence                              |
| `settings.go`    | Persisted settings (music on/off, playlist)                |
| `playlist.go`    | Playlist menu and enabled tracks                           |
| `demo.go`        | Attract-mode benchmark sequences                           |
| `synth.go`       | Oscillators, envelopes, filter                             |
| `tracker.go`     | Tracker song parser and adaptive sequencer                 |
| `composer.go`    | Procedural song composer and music styles                  |
| `audio.go`       | Audio engine and mixer                                     |
| `sounds.go`      | Procedural sound effect table                              |
| `game_audio.go`  | Music signals and sound triggers                           |

Entities are stored as struct of arrays with swap-remove, the spatial grid is rebuilt every tick with a counting sort, and every entity type is drawn in a single batched draw call.

## Extending

- New enemy: add a `EnemyKind` and an entry in `enemyTable`.
- New reaction: add a `ReactionKind`, its `reactionTable` entry, and a line in `reactionRules`.
- New ground rule: add a line in `groundRules`.
- New weapon: add a `WeaponKind`, a `weaponTable` entry and a behavior in `weaponBehaviors`.

## Tests and benchmarks

```sh
go test -run Long -v
go test -bench .
```

The long test runs 8 simulated minutes with every weapon maxed. `BenchmarkSimulateTenThousandEnemies` measures a tick with 10 000 live enemies packed around the player (about 5 ms on an i5-6200U).
