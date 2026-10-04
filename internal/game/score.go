package game

import (
	"fmt"

	"hordefall/internal/config"
)

// ===== Types =====

type HighScore struct {
	Score           int     `json:"score"`
	SurvivedSeconds float32 `json:"survivedSeconds"`
	Kills           int     `json:"kills"`
	Level           int     `json:"level"`
}

// ===== Constants =====

const (
	pointsPerExperience = 10
	pointsPerSecond     = 5
	highScoreFileName   = "highscore.json"
)

// ===== Public API =====

func (g *Game) CurrentScore() int {
	return g.killScore + int(g.elapsedSeconds)*pointsPerSecond
}

func LoadHighScore() (HighScore, error) {
	var highScore HighScore
	err := config.Load(highScoreFileName, &highScore)
	return highScore, err
}

func SaveHighScore(highScore HighScore) error {
	return config.Save(highScoreFileName, highScore)
}

func formatThousands(value int) string {
	digits := fmt.Sprint(value)
	grouped := make([]byte, 0, len(digits)+len(digits)/3)
	for index := range len(digits) {
		isGroupStart := index > 0 && (len(digits)-index)%3 == 0
		grouped = appendIf(grouped, ',', isGroupStart)
		grouped = append(grouped, digits[index])
	}
	return string(grouped)
}

// ===== Internal =====

func (g *Game) awardKillScore(experience int) {
	g.killScore += experience * pointsPerExperience
}

func (g *Game) loadHighScore() {
	highScore, err := LoadHighScore()
	if err != nil {
		return
	}
	g.highScore = highScore
}

func (g *Game) recordHighScore() {
	score := g.CurrentScore()
	if g.isDemo || score <= g.highScore.Score {
		return
	}
	g.highScore = HighScore{Score: score, SurvivedSeconds: g.elapsedSeconds, Kills: g.kills, Level: g.player.Level}
	g.isNewHighScore = true
	g.scoreMessage = ""
	if err := SaveHighScore(g.highScore); err != nil {
		g.scoreMessage = "High score not saved: " + err.Error()
	}
}
