package game

import "testing"

// ===== Tests =====

func TestRobotLabKeepsExactlyTwoEnemiesAndNoBoss(t *testing.T) {
	game := newHeadlessGame(t)
	game.robotLab = NewRobotLab()
	game.elapsedSeconds = 600
	game.bossProgressKills = game.bossKillsRequired + 1
	for range 600 {
		game.updateDirector(deltaSeconds)
		if game.enemies.Count != robotLabEnemyCount {
			t.Fatalf("%d enemies, want %d", game.enemies.Count, robotLabEnemyCount)
		}
	}
	game.enemies.Health[0] = 0
	game.removeDeadEnemies()
	game.updateDirector(deltaSeconds)
	if game.enemies.Count != robotLabEnemyCount {
		t.Fatalf("%d enemies after a kill, want %d", game.enemies.Count, robotLabEnemyCount)
	}
	if len(game.spiders) != 0 || len(game.worms) != 0 {
		t.Fatalf("a boss appeared in the lab")
	}
	for index := range game.enemies.Count {
		if game.enemies.Kind[index] >= EnemySpiderTank {
			t.Fatalf("enemy %d is a boss kind", index)
		}
	}
}

func TestRobotLabRestoresHealthEveryTick(t *testing.T) {
	game := newHeadlessGame(t)
	game.robotLab = NewRobotLab()
	game.player.Health = -500
	game.updateDirector(deltaSeconds)
	if game.player.Health != game.player.MaximumHealth {
		t.Fatalf("health %v, want %v", game.player.Health, game.player.MaximumHealth)
	}
}

func TestRobotLabSetsCycleThroughEveryRobot(t *testing.T) {
	game := newHeadlessGame(t)
	lab := NewRobotLab()
	game.robotLab = lab
	for index := 1; index <= len(robotLabSets); index++ {
		lab.selectNextSet(game)
		want := &robotLabSets[index%len(robotLabSets)]
		if game.robotLook != want.Look || lab.currentSetName(game.robotLook) != want.Name {
			t.Fatalf("step %d: wrong robot, want %s", index, want.Name)
		}
	}
	robotLabKeys[0].Advance(&game.robotLook)
	if lab.currentSetName(game.robotLook) != robotLabCustomName {
		t.Fatalf("a changed part must read as a custom mix")
	}
}

func TestEveryRobotPartDrawsOnScreenAndInTheArtwork(t *testing.T) {
	game := newHeadlessGame(t)
	canvas := NewArtCanvas()
	pose := zoomedPose(tsukumoArtX, tsukumoArtY, 1)
	for part := range 8 {
		for color := range robotColorCount {
			look := RobotLook{
				Head: RobotHead(part) % robotHeadCount, Eyes: RobotEyes(part) % robotEyesCount, Abdomen: RobotAbdomen(part) % robotAbdomenCount,
				Legs: RobotLegs(part) % robotLegsCount, Arms: RobotArms(part) % robotArmsCount, Color: color,
			}
			game.robotLook, canvas.Look = look, look
			rendererOnFreshBatches(game).queueTsukumo(game)
			canvas.beginLayer()
			canvas.paintRobot(pose)
			canvas.commitLayer()
		}
	}
}
