package game

import (
	"fmt"
	"slices"

	"hordefall/internal/config"
	"hordefall/internal/i18n"
)

// ===== Types =====

type HighScoreEntry struct {
	Initials        string  `json:"initials"`
	Score           int     `json:"score"`
	SurvivedSeconds float32 `json:"survivedSeconds"`
	Kills           int     `json:"kills"`
	Level           int     `json:"level"`
}

type HighScoreTable struct {
	Entries []HighScoreEntry `json:"entries"`
}

// ===== Constants =====

const (
	pointsPerExperience     = 10
	highScoreTableSize      = 10
	highScoreTableFileName  = "highscores.json"
	legacyHighScoreFileName = "highscore.json"
	legacyInitials          = "---"
	noRank                  = -1
	maximumScore            = 999999999
)

// ===== Public API =====

func (g *Game) CurrentScore() int {
	return min(g.killScore, maximumScore)
}

func (t *HighScoreTable) Best() HighScoreEntry {
	return t.EntryAt(0)
}

func (t *HighScoreTable) EntryAt(rank int) HighScoreEntry {
	if rank >= len(t.Entries) {
		return HighScoreEntry{}
	}
	return t.Entries[rank]
}

func (t *HighScoreTable) RankOf(score int) int {
	rank := len(t.Entries)
	for index, entry := range t.Entries {
		if score >= entry.Score {
			rank = index
			break
		}
	}
	isRanked := score > 0 && rank < highScoreTableSize
	return [2]int{noRank, rank}[boolToIndex(isRanked)]
}

func (t *HighScoreTable) Insert(entry HighScoreEntry) int {
	rank := t.RankOf(entry.Score)
	if rank == noRank {
		return noRank
	}
	t.Entries = slices.Insert(t.Entries, rank, entry)
	t.Entries = t.Entries[:min(len(t.Entries), highScoreTableSize)]
	return rank
}

func LoadHighScores() (HighScoreTable, error) {
	var table HighScoreTable
	if err := config.Load(highScoreTableFileName, &table); err == nil {
		clampHighScores(&table)
		return table, nil
	}
	var legacy HighScoreEntry
	err := config.Load(legacyHighScoreFileName, &legacy)
	legacy.Initials = legacyInitials
	table.Insert(legacy)
	return table, err
}

func SaveHighScores(table HighScoreTable) error {
	return config.Save(highScoreTableFileName, table)
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

func clampHighScores(table *HighScoreTable) {
	for index := range table.Entries {
		table.Entries[index].Score = min(table.Entries[index].Score, maximumScore)
	}
}

func (g *Game) awardKillScore(experience int) {
	g.killScore += experience * pointsPerExperience
}

func (g *Game) loadHighScores() {
	table, _ := LoadHighScores()
	g.highScores = table
}

func (g *Game) isScoreRanked() bool {
	return !g.isDemo && !g.isScoreRecorded && g.highScores.RankOf(g.CurrentScore()) != noRank
}

func (g *Game) recordHighScore() {
	g.recordHighScoreAs(g.settings.Initials)
}

func (g *Game) recordHighScoreAs(initials string) {
	if !g.isScoreRanked() {
		return
	}
	g.isScoreRecorded = true
	g.newEntryRank = g.highScores.Insert(HighScoreEntry{
		Initials: initials, Score: g.CurrentScore(), SurvivedSeconds: g.elapsedSeconds, Kills: g.kills, Level: g.player.Level,
	})
	g.saveHighScores()
}

func (g *Game) saveHighScores() {
	g.scoreMessage = ""
	if err := SaveHighScores(g.highScores); err != nil {
		g.scoreMessage = i18n.F("scores.save_failed", err.Error())
	}
}

func (g *Game) finishRun() {
	if g.isScoreRanked() {
		g.openNameEntry()
		return
	}
	g.switchState(StateGameOver)
}
