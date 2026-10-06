package game

import (
	"testing"

	"hordefall/internal/i18n"
)

func TestChatterEntersRightAndLeavesLeftPixelByPixel(t *testing.T) {
	boxWidth, textWidth := float32(160), float32(300)
	if marqueeOffset(0, boxWidth) != boxWidth {
		t.Fatal("the line does not start beyond the right edge")
	}
	step := marqueeOffset(0.5/chatterScrollSpeed, boxWidth)
	if step != boxWidth {
		t.Fatalf("the line moved by a fraction of a pixel: %v", step)
	}
	if marqueeOffset(1.5/chatterScrollSpeed, boxWidth) != boxWidth-1 {
		t.Fatal("the line does not advance one whole pixel at a time")
	}
	chatter := Chatter{Runes: []rune("x"), Seconds: (textWidth + boxWidth) / chatterScrollSpeed * 0.99}
	if !chatter.IsSpeaking(textWidth, boxWidth) {
		t.Fatal("the line stopped before leaving the left edge")
	}
	chatter.Seconds = (textWidth + boxWidth) / chatterScrollSpeed * 1.01
	if chatter.IsSpeaking(textWidth, boxWidth) {
		t.Fatal("the line is still shown after leaving")
	}
}

func TestChatterIsDebounced(t *testing.T) {
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	i18n.Use(i18n.FallbackCode)
	game := newHeadlessGame()
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
