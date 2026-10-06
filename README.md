# Hordefall

A systemic horde survivor written in Go with [Ebitengine](https://ebitengine.org).
Thousands of enemies, auto-firing elemental weapons, and a living ground: every element interacts with enemies, with the terrain, and with each other.

## Run

```sh
go run .
```

## Menus, score and high score

- **Main menu**: Play, Options, Quit, and your high score (dark orange).
- **Options**: Controls, Gameplay (High scores, to view the ranking; Reset high scores, behind a No / Yes confirmation with No selected), Graphics (Fullscreen, also F11 anywhere; VSync; Resolution 720p / 1080p / Native, where Native renders at the screen's real pixel count for sharp lines and text; Show FPS; Screen shake Off / 50% / 100%; Effects Low / Medium / High, which keeps 30% / 60% / 100% of particles; Bloom Off / Soft / Strong, a glow on the world only (fire, lightning, explosions), never on the HUD; CRT filter, with curved screen, scanlines, RGB mask, colour fringing and vignette), Audio, a submenu with Music on/off (sound effects stay on), Music volume and Effects volume, each from 0 to 100% in 20% steps (Left / Right on its gauge, or Fire to step up and wrap), and Playlist (switch each track on or off: enabling a track plays it at once, disabling the current one moves to the next enabled track, rotation goes to the next enabled track; all off means no music), Back. Settings are saved to `%AppData%\hordefall\settings.json` and restored on launch. Defaults on a fresh install: windowed, VSync off, Native resolution, FPS hidden, screen shake 50%, effects High, bloom Soft, CRT on, music volume 60%, effects volume 80%, every track enabled; the playlist is a `"tracks": { "title": true/false }` map, and a track missing from it counts as enabled. Behind it runs an attract-mode benchmark: eight 5-second gameplay sequences on loop (Inferno chain, firing only half the time; Electrocution; Freeze and shatter; Wildfire; Spider tank; Horde x4000; Siege: 3 spider tanks and a horde of 2000; Full arsenal: all 6 weapons at max level against 2500 soaked foes) interleaved with randomly generated LIVE ACTION clips (random weapons and levels, passives, enemy mix and count, starting statuses, prepared ground, sometimes a spider tank), all played by an autopilot Tachikoma; whenever a chain reaction leaves fewer than 40% of the enemies, a fresh wave without status walks in from the edge so no clip runs empty; with live FPS, TPS, simulation time and entity counts. After every full rotation of the demo (all 16 clips), an arcade interlude takes over (any button closes it): a black screen, a double flash, then a 128x72 artwork, zooming in from the distance, of the Tachikoma rolling toward the camera while a towering spider tank leans in to intercept it, and after 2 seconds the RANKING board scrolls over it for 10 seconds.
- **Score**: every kill is worth its experience x 10 (a spider tank is worth 1,500, a sandworm 2,500), and a cleared horde adds 20 per member. Survival time gives no points: the score measures skill. Score and high score are shown in the HUD.
- **Ranking**: a top 10 sorted by score, saved to `%AppData%\hordefall\highscores.json` (an old single `highscore.json` is imported as `---`). Every run ends with the arcade initials entry, starting at AAA (titled NEW HIGH SCORE!, GREAT SCORE! with the rank, or GAME OVER when the score is not ranked, in which case nothing is recorded): you enter 3 initials (Up / Down: letter, held for 2 seconds it scrolls fast; Left / Right: move, Fire: confirm, Aim lock: back); after confirming, the game over screen shows your ranked line in the pixel font, blinking, above the Start / Select choices; your last initials are reused when a ranked run is left from the pause menu or by closing the window. The board shows the game title, RANKING and columns RANK, SCORE, TIME, NAME in a blocky 5x7 pixel font (`pixel_font.go`), half of it on screen: it pauses, scrolls to the bottom, pauses again.
- **Pause** (Start / Esc) freezes everything, including screen shake and fire flicker: Resume, Options (the same options as the main menu; the run stays frozen behind them and Back returns to the pause menu), Music on/off, Back to main menu. In every menu, Start / Esc or the aim lock button (button 2 / K) goes back, or resumes from pause; the button setup screen keeps button 2 free to be assigned.
- **Game over**: START to retry, SELECT for the main menu. The music plays with every layer in. Left idle for 20 seconds, the screen moves on to the RANKING interlude, then the title. Button names follow the last device used: keyboard keys (ENTER, BACKSPACE) or gamepad buttons, including the buttons you configured.
- **Level up**: when several upgrades are available the game pauses on a choice of cards, the highlighted one framed in a thick red border; when only one is possible it is applied at once, with a LEVEL UP! banner and no interruption. Once every weapon is maxed and only passives remain, each level up applies the next remaining passive in turn (a b c d a b c d, then a b d a b d once c is maxed), also without pausing.
- **Heal bubbles**: each enemy you kill has a 2% chance to also drop a green bubble with a white cross. It gives no experience but restores 20 health (up to your maximum), and it is drawn in by your pickup radius like experience gems (`heal_orbs.go`). Enemies killed by a boss drop none.

## Music and sound

### Adaptive tracker music
The soundtrack is a small Amiga-style tracker running live: `internal/audio/music/theme.trk` is a plain-text song with 16 channels, patterns of rows, an order list and synthesized instruments (square with duty cycle, saw, triangle, sine, noise; ADSR envelope, pitch slide, low-pass filter). Effects: `0xy` arpeggio (the classic Amiga chord shimmer) and `Cxx` volume.

The music is procedural in the sense that it follows the game. Every channel always plays in time, but is only heard while its game signal is above a threshold, with a smooth fade:

| Layer | Signal | Meaning |
|-------|--------|---------|
| drums, bass | `always` | the base groove |
| hi-hats, chords, lead | `horde` | enemies massing around you |
| boss voices | `boss` | one deep voice per spider tank alive: saw stabs on the root (1 tank), a lower square growl on the fifth (2 tanks), the deepest saw drone sliding a semitone up and a tone down (3 tanks) |
| crystal bells | `reactions` | infernos, freezes, shatters chaining |
| tension pulse | `danger` | low health or a spider beam charging |
| shaker, pad | `always` | 16th-note shaker in the gaps of the hi-hats, a soft sustained chord in the mids |
| clap | `horde` 0.2 | doubles the snare, or marks the backbeat when the kick hides it |
| pluck arpeggio | `horde` 0.3 | climbs the chord tones (every 8th note, every quarter in slow songs) |
| offbeat stabs | `horde` 0.5 | short chords between the kicks |
| lead harmony | `horde` 0.7 | a second voice on the nearest chord tone under the melody |

Channels 11 to 16 are written by the arranger (`arranger.go`) for every song, the theme and the composed ones alike: it reads each pattern's chord from the arpeggio notes on channel 4 and its rhythm from the drums and hi-hats, so every added note is a chord tone and every added hit falls between or on the existing ones. A test checks this for each song. Write cells in one of those channels in `theme.trk` and the arranger leaves that channel alone.

The tempo also rises with intensity (`tempoboost`). M mutes the music.

### Composing
Edit `internal/audio/music/theme.trk` and rebuild: it is embedded in the executable. A row is up to 16 cells separated by `|` (missing channels are empty), each cell is `NOTE INSTRUMENT EFFECT`, for example `A-4 05 037` (A4, instrument 5, minor-chord arpeggio). `---` means nothing, `===` releases the note, `..` and `...` leave instrument and effect empty. Instruments, layers (`layer <channel> <signal> <threshold>`), tempo and pan are declared at the top of the file. Pan uses a constant-power law (0 left, 0.5 centre, 1 right): kick, bass and boss voices stay centred so the low end is balanced in headphones, while hi-hats, chords, lead, bells and the danger pulse are spread. The music bus then goes through a small stereo room (`stereo_room.go`: two different delays crossed left and right, with the bass filtered out) for width without lopsided panning. Parse errors give the line and channel, and `go test ./internal/audio -run Theme` checks the song.

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
$env:HORDEFALL_WAV_DIR = "./music/previews"; go test ./internal/audio -run ExportSongPreviews
```

### Procedural sound effects
Every sound effect is synthesized on the fly from layered voices with a random pitch variation, so no two pops are identical: enemy pops, thuds and splashes, weapon pews, zaps and novas, reaction explosions, shatters, hisses and electrocutions, spider stomps, beam charge and fire, dash, hurt, level-up arpeggio, gem pickup and menu blips. Each sound has a minimum interval, so a hundred kills in a frame never saturate the mix. Sounds are declared in `sounds.go`; enemies, reactions and weapons reference them from their own tables.

## Controls

| Action        | Arcade stick / gamepad  | Keyboard        | Mouse |
|---------------|-------------------------|-----------------|-------|
| Move          | Stick (D-pad or analog) | WASD / arrows   |       |
| Aim           | Right stick (twin-stick pads) | follows movement | cursor around the Tachikoma |
| Fire (hold)   | Button 1                | J / Z           | left button |
| Aim lock (hold) | Button 2              | K / X           |       |
| Dash          | Button 3                | Space / L / C   | right button |
| Pause         | Start                   | Esc / P         |       |
| Live benchmark on / off (title) | Select | Backspace / F2 |      |
| Input debug (in game) | Select          | F1              |       |

Mouse aiming switches on as soon as the mouse moves or clicks: the cannon points at the cursor, drawn as a crosshair while the system cursor is hidden in play. Touching the gamepad hands the aim back to it; a keyboard-only player who never touches the mouse keeps the aim that follows movement.

Aim follows your movement. Hold aim lock to freeze it and strafe or retreat while firing.
On a gamepad with two sticks, the right stick aims (twin-stick): move with the left stick, aim with the right one; pushing it past the deadzone also fires automatically, so no button is needed; release it and the aim follows your movement again. Fire still works on button 1, which is how the arcade stick plays.
Ember Bolt, Arc Lightning, Oil Flask and Downpour fire where you aim while Fire is held. Frost Nova, Orbit Blades, Seismic Hammer and Static Mines are automatic.

Every control can be remapped, separately for the keyboard and the gamepad: the Controls screen (Options > Controls, also from the pause menu) sets up the device you opened it with.
- The first time for a device, it asks for Fire, Aim lock and Dash in turn (Start / Esc cancels).
- Then it lists every action with its key or button: Fire, Aim lock, Dash, Pause, Select, plus Up, Down, Left and Right on the keyboard. Choose a line with Up / Down and press Fire: the line listens for 5 seconds and takes the next key or button. A key already used by another action swaps places with it. Reset to defaults restores the device.
- Arrows, Enter, F1, M and F11 stay reserved so the menus always work; the gamepad D-pad and sticks always move. A gamepad button that still holds another action's default (Start, Select, A, B, X) is refused until that action is remapped.
- The mapping is saved to `%AppData%\hordefall\controls.json` (`keys` and `buttons`, -1 meaning the default) and loaded on startup; the old Fire / Aim lock / Dash file is imported. Button names shown in the game follow the last device used.

## The systems

**Statuses** (`status.go`): Burning, Chilled, Frozen, Oiled, Wet, Shocked. Each is a bit in `StatusFlags` with a duration, tint, speed factor and damage over time. Burning, Chilled and Shocked stack up to level 3 with repeated hits: the tint grows stronger (and hotter for fire, from red to glowing yellow), a light aura appears from level 2 and a 1 px rim of light at level 3. Burning deals 6 / 12 / 18 per second, Chilled and Shocked slow more at each level, and the third frost hit freezes.

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
| Fire     | Shocked          | PLASMA       | x1.8, sets ablaze and leaps to every shocked enemy nearby |
| Physical | Shocked          | OVERLOAD     | x2.5 and jolts the shocked enemies around      |
| Water    | Shocked          | CONDUCTION   | The charge spreads to every wet enemy nearby (then ELECTROCUTE) |

Reaction bursts carry an element too, so reactions chain into further reactions.

**Ground** (`ground.go`): a 256x256 cellular grid. Grass catches fire and spreads it, oil burns explosively, ice melts, water freezes under frost, ash regrows into grass. Standing on a cell applies its element to enemies (fire burns, water wets, oil soaks, ice chills) and fire hurts the player.

**Weather** (`weather.go`): every 3 to 4 minutes, a 30 second weather event changes the whole map, with its own color grading and particles:

| Weather  | Enemies                      | Ground                         | Extra                         |
|----------|------------------------------|--------------------------------|-------------------------------|
| STORM    | Soaked (Wet), fires put out  | Burning cells are doused       | Lightning strikes shock the area |
| HEATWAVE |                              | Grass catches fire everywhere  | Warm grading, rising embers   |
| BLIZZARD | First level of Chilled       | Water freezes, fires die out   | Cold grading, snow            |

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

## The talkative Tachikoma

Like its namesakes in Stand Alone Complex, the Tachikoma chats: childlike, curious, a little philosophical (does a tank have a ghost?), fond of natural oil and of Batou-san. Its lines scroll on a teletext strip at the bottom of the HUD, white on a half-opaque panel in a fixed-width font: the first 15 columns show, then after 2 seconds the line scrolls one character at a time to its end, holds and fades, with a little robot chirp (`chatter.go`).

- It speaks when a run starts, on level up, heal bubbles, spider tanks and the sandworm arriving or falling, hordes, big chain reactions, low health, weather, a shot evolution, or after 50 quiet seconds.
- It never nags: at least 12 seconds between any two lines, a minute between level up or heal lines, 20 to 45 seconds for the other topics, and never twice the same line in a row.
- Rival lines (another Tachikoma too close, a stolen kill, being overtaken in level) are written for a future multiplayer mode and not triggered yet.
- All 50 lines are translated in the nine languages (`tachikoma.<topic>.<n>` keys); Japanese, Chinese and Korean characters count as two columns.

## Boss: Spider Tank

Boss events follow a cycle: three spider tanks, a horde, the sandworm, a horde, then again. Each boss event needs its own kills: the counter resets when a boss arrives and starts counting again at once, so the next event can come quickly, even while a boss is still alive. The requirement grows with the run: 300 kills at the start, 300 × (1 + minutes / 2) after that (1,050 at 5 minutes, 1,800 at 10, 3,300 at 20). A Ghost in the Shell style spider tank then walks in. At most 3 bosses (spider tanks and sandworm) can be alive at once, so their health gauges stay readable; when the cap is reached the cycle waits. The horde is the exception and never counts toward it.
Its four legs are procedural: each leg is a two-bone chain solved with inverse kinematics (the knee is the outward solution of the two-circle intersection), each foot stays planted until it drifts too far from its rest pose, then steps ahead of the body with an eased arc. Legs move in diagonal pairs.
Every footfall is a stomp: dust, screen shake, a physical burst that crushes nearby enemies, shatters frozen ones and hurts you.
Its cockpit, sensor and cannon sit on a turret ring that tracks you with a slow, heavy servo, independently of where the legs carry the body.
**Devastator beam.** It spawns off screen and opens with its signature attack, so you know it is there:
1. Charging (2.2 s): glowing white motes are sucked into the front of the cannon, where a white sphere grows, while a thin, intense targeting beam leaves the cannon and tracks you through the turret servo.
2. Locked (400 ms): the beam freezes and blinks, and the full sphere radiates lightning that ricochets off the ground around it. This is your window to get out, or to dash through.
3. Firing (0.35 s): a long, thick, blinding ray. It shreds your health (dashing makes you immune), burns every enemy it crosses, and scorches the ground: grass and oil catch fire, ice melts, dirt turns to ash.
4. Cooldown (6 to 9 s), then again.
Enemies killed by a boss give no experience, no kill and no lifesteal. Timings, damage and range live in `spiderBeamProfile` in `boss_beam.go`.

It takes statuses and reactions like any enemy, and explodes into a firestorm when destroyed.
Tuning lives in the constants and `spiderLegLayouts` of `spider.go`; drawing is in `spider_render.go`.

## Boss event: the Horde

After three spider tanks, the fourth boss event is a horde closing in from every side: 500 the first time, then 1,000, 1,500 and so on. Its core is a mix of brutes, frostlings, runners and swarmers, all twice as tough as usual; the explosive enemies (bloaters and emberlings, one in eight) form the outer shell, so the horde no longer goes up in one chain reaction. It counts as a boss: its purple bar shows how many are left and the music adds a boss voice. Other bosses keep coming meanwhile, and a horde arriving while one is still alive joins it. Horde kills give experience and score but do not advance the boss counter. Clearing it awards a bonus of 20 points per member. Tuning in `horde_event.go`.

## Boss: the Sandworm

A robotic worm of fourteen armored links and a heavy head, built like the multi-jointed bosses of Treasure or the Gradius tentacles: the head leads, turning toward you at a limited rate, and every link copies the head's orientation a few frames later (3 frames per link), straightening toward the vertical near its crater, so the body is laid out link after link from the ground like a metal chain and a wave runs down it when the head turns. Links and head are pixel-art sprites generated at launch in 32 orientations (1 px black outline, 6-shade dithered metal lit from the top left, rings, rivets, cyan vents, red sensors, mandibles), drawn without smoothing and scaled in 1/8 steps with their height, the Neo-Geo way: the higher a link, the closer and bigger it is, and higher links are drawn over lower ones.
It comes after the first horde of each cycle, and only one can be alive at a time: when its turn comes back while it still lives, the cycle waits for it. It never holds the other events back, so spider tanks and the horde that follows it keep coming, and it arrives alongside spider tanks as long as fewer than 3 bosses are alive.
1. Burrowing (1.6 to 3.5 s): a small cracked star of earth quivers above it while it closes in under the ground, leaving a wake of dirt clods and brown dust. It cannot be hit.
2. Warning (1.1 s): once under you, the cracked star grows under your position, pulsing red, with a deep rumble. Move away.
3. Emergence: the ground bursts with a roar and leaves a pixel-art crater, a star of earth with many irregular branches around a smaller black star (four variants generated at launch), under a cloud of brown dust; it fades after 8 s. The blast crushes enemies, and if you are caught it throws you back and hits you five times in quick succession without ever taking your last hit point. The worm rises straight up out of a rocky crater as a column, then rears, its head turning toward you and the body bending after it.
4. Aiming: the head keeps tracking you (1.4 rad/s) and the beam can only leave along the head, so it misses while the head lags behind you. It charges a frost beam, a slower version of the spider tank's devastator: 3.4 s of charge and 1 s of lock (the beam freezes and blinks), then the same instant 0.35 s shot, twice as wide. It chills every enemy it crosses and freezes the ground. After the shot it waits 1.4 s, then dives in an arc: the head plunges forward into a new crater and the whole body follows the same curve underground, then it burrows again.
Each segment has its own small health gauge and can only be hit while above ground. Damaged segments spark below 75%, smoke below 50% and burn below 25%, then explode, and the body closes up, shorter. Once every segment is destroyed, the head breaks free and stays out: every 4 seconds it aims a thin red laser at you for 1.2 s (locked during the last 0.3 s), then charges along it with inertia, rolling 720 px straight ahead, crushing enemies and slamming you if you are on its path. Destroying the head kills the worm.
The beam is shared with the spider tank through a `BeamProfile` (timings, width, damage, status, ground effect, sounds, colours) in `boss_beam.go`. Tuning in `worm.go`, sprites in `worm_sprites.go`, drawing in `worm_render.go`.

## Weapons

You start with **Pulse Shot**, a neutral bolt. At level up it can evolve, keeping its level, into **one** elemental shot: Ember Bolt (fire), Frost Shard (frost) or Volt Bolt (shock). Once one is chosen, the others are no longer offered, only its upgrades.

Ember Bolt (fire), Frost Nova (frost, freezes water), Arc Lightning (shock chains), Oil Flask (soaks ground and enemies), Downpour (wets, douses fires), Orbit Blades (physical, shatters frozen), Seismic Hammer (physical slam with knockback; its wide quake shatters every frozen enemy far around), Static Mines (dropped at your feet, they shock whoever steps near; shocked enemies turn fire into PLASMA, blades and slams into OVERLOAD, rain into CONDUCTION). The last two keep fire builds whole: neither puts flames out. Up to 5 weapons, level 7 each, plus 6 passives.

## Architecture

```
main.go                   entry point
internal/
  game/                   the game: state machine, simulation, entities, bosses, UI, rendering
  audio/                  synth, tracker, composer, arranger, sound effects, music/theme.trk
  config/                 settings, controls and high score files
  spatial/                neighbor-query grid
  rng/                    random generator
```

Go builds one package per folder and a type's methods must live in its own package. The `Game` type and its methods therefore stay together in `internal/game`; subsystems with a clean boundary live in their own packages and expose a small API (`audio.Engine`, `config.Load` / `config.Save`, `spatial.Grid`, `rng.Random`).

### internal/game

| File | Role |
|------|------|
| `audio_menu.go` | Audio submenu: music toggle, volume gauges, playlist entry |
| `attract_art.go` | Lo-res artwork rasterized each frame: shaded ellipsoids and capsules, outlines, coloured rim lights |
| `bindings.go` | Remappable actions, defaults, swaps and persistence |
| `remap.go` | Controls screen: essential sequence, list, listening, reset |
| `chatter.go` | Talkative Tachikoma: topics, cooldowns, teletext strip |
| `combat.go` | Hits, reactions, area bursts, deaths |
| `demo.go` | Attract-mode benchmark sequences |
| `effects.go` | Particles, rings, lightning, popups, screen shake |
| `enemies.go` | Enemy table and struct-of-arrays store |
| `game.go` | Game loop, state machine table |
| `game_audio.go` | Music signals and sound triggers |
| `gems.go` | Experience gems |
| `graphics.go` | Graphics options, render scale, scaled text and rectangles |
| `ground.go` | Living terrain cellular automaton |
| `horde.go` | Enemy movement, separation, contagion, spawn director |
| `horde_event.go` | Horde boss event: spawning, tracking, bar |
| `input.go` | Keyboard and gamepad bindings, raw button capture |
| `mathutil.go` | Small math helpers |
| `menus.go` | Main menu, pause, game over and remap states |
| `menus_test.go` | Menus, settings, playlist and song rotation tests |
| `heal_orbs.go` | Heal bubbles: 2% drop, pickup, +20 health, drawing |
| `mines.go` | Static mines: drop, arming, trigger, rendering |
| `player.go` | Player movement, dash, damage, experience |
| `playlist.go` | Playlist menu and enabled tracks |
| `postfx.go` | Kage shaders: bloom (bright pass, blur) and CRT filter |
| `projectiles.go` | Bolts and lobbed flasks |
| `render.go` | Batched circle sprites with `DrawTriangles32`, additive glow |
| `score.go` | Score, high score persistence |
| `settings.go` | Persisted settings (music on/off, playlist) |
| `simulation_test.go` | Simulation, balance, bosses and demo tests |
| `skids.go` | Wheel skid trails |
| `spider.go` | Spider tank boss: spawn, procedural legs, gait, stomps |
| `spider_charge.go` | Beam charge visuals: drawn-in motes, sphere, discharge |
| `boss_beam.go` | Boss beams (spider tank, sandworm): charge, lock, fire state machine driven by a `BeamProfile` |
| `worm.go`, `worm_render.go`, `worm_sprites.go` | Sandworm boss: burrowing, chain body, frost beam, destructible links, rolling head, pixel sprites |
| `spider_render.go` | Spider tank drawing |
| `status.go` | Statuses, elements, reaction table |
| `tachikoma.go` | Player rig: wheel springs, abdomen sway, gaze, recoil |
| `tachikoma_render.go` | Player drawing |
| `ui.go` | HUD, menus, overlays |
| `upgrades.go` | Passives and level-up offers |
| `weapons.go` | Weapon table and behaviors |
| `intro.go`, `intro_logo.go`, `intro_glass.go` | Studio intro: voxel GREG's logo, tumbling fall, shattered glass |
| `language.go` | Language row with its drum, first launch page, i18n loading |
| `gameplay_menu.go` | Gameplay submenu, high score reset confirmation, scrolling RANKING board |
| `name_entry.go` | Arcade initials entry after a ranked run |
| `pixel_font.go` | Blocky 5x7 pixel font drawn as batched square pixels |
| `weather.go` | Weather cycle, enemy and ground effects, lightning, precipitation, color grading |

### internal/audio

| File | Role |
|------|------|
| `arranger.go` | Arranges channels 11-16 from each pattern's chords |
| `audio_test.go` | Tracker, composer, arranger, stereo and sound effect tests |
| `composer.go` | Procedural song composer and music styles |
| `engine.go` | Audio engine and mixer |
| `mathutil.go` | Small math helpers |
| `sounds.go` | Procedural sound effect table |
| `stereo_room.go` | Stereo room on the music bus: crossed delays, no bass |
| `synth.go` | Oscillators, envelopes, filter |
| `tracker.go` | Tracker song parser and adaptive sequencer |

### Other packages

| Package | File | Role |
|---------|------|------|
| `config` | `store.go` | JSON files in %AppData%\hordefall: path, load, save |
| `spatial` | `grid.go` | Uniform grid with counting sort for neighbor queries |
| `rng` | `random.go` | Xorshift random generator |
| `i18n` | `i18n.go`, `locale*.go`, `lang/*.json` | Language files, English fallback, external `lang` folder, system language detection |

Entities are stored as struct of arrays with swap-remove, the spatial grid is rebuilt every tick with a counting sort, and every entity type is drawn in a single batched draw call.

## Extending

- New enemy: add a `EnemyKind` and an entry in `enemyTable`.
- New reaction: add a `ReactionKind`, its `reactionTable` entry, and a line in `reactionRules`.
- New ground rule: add a line in `groundRules`.
- New weapon: add a `WeaponKind`, a `weaponTable` entry and a behavior in `weaponBehaviors`.

## Tests and benchmarks

```sh
go test ./...
go test ./internal/game -run Long -v
go test ./internal/game -bench .
```

The long test runs 8 simulated minutes with every weapon maxed. `BenchmarkSimulateTenThousandEnemies` measures a tick with 10 000 live enemies packed around the player (about 5 ms on an i5-6200U).

## Release builds

Double-click `build.bat` on Windows, or run `./build.sh` (or `go run ./tools/release`) anywhere. Each platform is compiled and streamed straight into its own archive in `dist/`, named after the git version:

| Target | Archive | Built from |
|--------|---------|------------|
| `windows-amd64`, `windows-arm64` | `.zip` with `Hordefall.exe` (no console window) | any OS |
| `macos-universal` | `.tar.gz` with `Hordefall.app`, one binary for Apple Silicon and Intel, macOS 12+ | any OS |
| `linux-amd64` | `.tar.gz` | Linux, or any OS with Docker running |
| `freebsd-amd64` | `.tar.gz` | FreeBSD |

Ebitengine needs cgo and the system graphics libraries on Linux and FreeBSD, so those targets are skipped, with the reason printed, when they cannot be built from the current machine. Pick targets with `-targets windows-amd64,macos-universal`. The Mac app is not signed: on first launch, right-click it and choose Open, or run `xattr -cr Hordefall.app`.

## Fonts

The interface uses Rajdhani (Medium for text, Bold for titles and menus), by the Indian Type Foundry, under the SIL Open Font License (`internal/game/fonts/OFL.txt`). It covers Latin and Devanagari. Other scripts fall back, glyph by glyph, to Exo 2 for Cyrillic and to Noto Sans KR, JP and SC for Korean, Japanese and Chinese (both under the SIL Open Font License, `OFL-Exo2.txt` and `OFL-NotoSansCJK.txt`), then to the Go fonts (the music note). The order of the three CJK fonts follows the language, so Japanese and Chinese characters take their own shapes.

The fallback fonts are subsets holding only the characters of the language files, about 1.1 MB in all. After editing a language file, rebuild them with `py -3 tools/fonts/subset.py` (Python with fontTools; the full fonts are downloaded once into `tools/fonts/cache`, which git ignores).

## Languages

English, French, Spanish, Russian, Serbian (Cyrillic), Korean, Japanese, Simplified Chinese and Hindi, in `internal/i18n/lang/<code>.json`: `{"code", "name", "script", "strings": {key: text}}`. Texts that arcade games keep in English stay in English: the title, RANKING and its columns, HI and SCORE, LEVEL UP, NEW HIGH SCORE!, the initials entry, reaction and demo clip names, FPS and TPS, button names.

- The game opens on a short studio intro: the word GREG's, built from chunky orange pixels in 3D, tumbles out of the dark toward the screen to a falling-bomb whistle, lands flat and shatters the screen into smooth glass cracks with a boom; it fades to the menu after a pause, and three presses of Fire skip it.
- On the first launch the game reads the system language (Windows, macOS, Linux), preselects it when it is available, or English otherwise, and waits for the player to confirm. The choice lives in `settings.json` as `language` (`default` until confirmed, which means English); it can be changed in Options > Gameplay on the Language line: Fire turns its value into a small drum that rolls with Up / Down (it wraps, and scrolls by itself after 2 seconds held), previews each language live, and Fire confirms or Back cancels.
- A `lang` folder next to the game (or in the working directory) is read at launch: a new file adds a language to the list, a file with an existing code overrides its texts.
- Nothing ever breaks on a missing piece: a missing key falls back to English, then to the key itself; a broken file is skipped; a missing glyph falls back to the next font.
- Tests check that every key the code uses exists in English, that translations keep the same `%` placeholders, that every character of every language has a glyph, and that every translation fits its space (titles, menus, hints, two lines per upgrade card): when one overflows, reword it shorter.
