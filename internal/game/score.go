package game

import "github.com/ajkula/ajkurig/scores"

// ===== Constants =====

const pointsPerExperience = 10

// ===== Public API =====

func (g *Game) CurrentScore() int {
	return min(g.killScore, scores.MaximumScore)
}

// ===== Internal =====

func (g *Game) awardKillScore(experience int) {
	g.killScore += experience * pointsPerExperience
}
