package game

import (
	"testing"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Constants =====

var cardFitRules = []ui.FitRule{
	{Prefix: "weapon.", Suffix: ".description", Face: ui.FaceRegular, MaxWidth: ui.CardTextWidth, MaxLines: 2},
	{Prefix: "passive.", Suffix: ".description", Face: ui.FaceRegular, MaxWidth: ui.CardTextWidth, MaxLines: 2},
	{Prefix: "weapon.", Face: ui.FaceBold, MaxWidth: ui.CardTextWidth, MaxLines: 1},
	{Prefix: "passive.", Face: ui.FaceBold, MaxWidth: ui.CardTextWidth, MaxLines: 1},
	{Prefix: "main.controls", Face: ui.FaceSmall, MaxWidth: 1240, MaxLines: 1},
}

// ===== Public API =====

func TestTranslationsFitTheirSpace(t *testing.T) {
	app, _ := newTestApp(t)
	rules := append(append([]ui.FitRule{}, cardFitRules...), ui.DefaultFitRules...)
	for _, overflow := range app.Kit.Text.FindOverflows(i18n.Languages(), rules) {
		t.Errorf("does not fit: %s", overflow)
	}
}
