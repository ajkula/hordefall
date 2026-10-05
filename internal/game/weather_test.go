package game

import "testing"

func TestWeatherStartsAfterCalmAndEnds(t *testing.T) {
	game := newHeadlessGame()
	game.weather.CalmSeconds = deltaSeconds
	game.updateWeather(deltaSeconds)
	if game.weather.Kind == WeatherClear {
		t.Fatal("weather did not start after the calm")
	}
	for range weatherActiveSeconds*ticksPerSecond + 1 {
		game.updateWeather(deltaSeconds)
	}
	if game.weather.Kind != WeatherClear || game.weather.CalmSeconds < weatherCalmMinimumSeconds-1 {
		t.Fatalf("weather did not end: %+v", game.weather)
	}
}

func TestBlizzardOnlyAppliesFirstChillLevel(t *testing.T) {
	game := newHeadlessGame()
	for index := range 200 {
		game.enemies.Spawn(EnemyKind(0), game.player.X+float32(index), game.player.Y+300, 1)
	}
	game.weather = Weather{Kind: WeatherBlizzard, ActiveSeconds: weatherActiveSeconds / 2}
	for range 10 * ticksPerSecond {
		game.updateWeather(deltaSeconds)
	}
	chilledIndex := statusBitIndex(StatusChilled)
	chilled := 0
	for index := range game.enemies.Count {
		chilled += boolToIndex(game.enemies.Status[index]&StatusChilled != 0)
		if game.enemies.StatusLevels[chilledIndex][index] > 1 || game.enemies.Status[index]&StatusFrozen != 0 {
			t.Fatalf("enemy %d went past the first chill level", index)
		}
	}
	if chilled < game.enemies.Count/2 {
		t.Fatalf("only %d of %d enemies chilled", chilled, game.enemies.Count)
	}
}

func TestStormSoaksEnemiesAndPutsOutFire(t *testing.T) {
	game := newHeadlessGame()
	for index := range 200 {
		game.enemies.Spawn(EnemyKind(0), game.player.X+float32(index), game.player.Y+300, 1)
		game.enemies.ApplyStatus(index, StatusBurning)
	}
	game.weather = Weather{Kind: WeatherStorm, ActiveSeconds: weatherActiveSeconds / 2}
	for range 8 * ticksPerSecond {
		game.updateWeather(deltaSeconds)
	}
	soaked := 0
	for index := range game.enemies.Count {
		isSoaked := game.enemies.Status[index]&StatusWet != 0
		soaked += boolToIndex(isSoaked)
		if isSoaked && game.enemies.Status[index]&StatusBurning != 0 {
			t.Fatalf("enemy %d is wet and still burning", index)
		}
	}
	if soaked < game.enemies.Count/2 {
		t.Fatalf("only %d of %d enemies soaked", soaked, game.enemies.Count)
	}
}
