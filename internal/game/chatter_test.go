package game

import (
	"testing"

	"github.com/ajkula/ajkurig/i18n"
)

func TestChatterIsDebounced(t *testing.T) {
	if err := loadGameLanguages(); err != nil {
		t.Fatal(err)
	}
	i18n.Use(i18n.FallbackCode)
	game := newHeadlessGame(t)
	game.chatter.QuietSeconds = chatterGlobalGap
	game.say(ChatterHeal)
	if len(game.chatter.Runes) == 0 {
		t.Fatal("the first heal line was not said")
	}
	first := string(game.chatter.Runes)
	game.chatter.QuietSeconds = chatterGlobalGap
	game.say(ChatterHeal)
	if string(game.chatter.Runes) != first || game.chatter.QuietSeconds != chatterGlobalGap {
		t.Fatal("a second heal line was said within its cooldown")
	}
	game.chatter.QuietSeconds = 1
	game.say(ChatterSpiderInbound)
	if string(game.chatter.Runes) != first {
		t.Fatal("a line was said within the global gap")
	}
	game.isDemo = true
	game.chatter.QuietSeconds = chatterGlobalGap
	game.say(ChatterSpiderInbound)
	if string(game.chatter.Runes) != first {
		t.Fatal("the demo talked")
	}
}

func TestBossLinesCutTheGlobalGapShort(t *testing.T) {
	if err := loadGameLanguages(); err != nil {
		t.Fatal(err)
	}
	i18n.Use(i18n.FallbackCode)
	game := newHeadlessGame(t)
	game.chatter.QuietSeconds = chatterUrgentGap
	game.say(ChatterHeal)
	if len(game.chatter.Runes) != 0 {
		t.Fatal("an ordinary line skipped the global gap")
	}
	game.say(ChatterWormBlast)
	if len(game.chatter.Runes) == 0 {
		t.Fatal("the drill-worm blast line waited for the global gap")
	}
}
