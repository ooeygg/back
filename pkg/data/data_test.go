package data

import (
	"testing"

	"github.com/ooeygg/remas/back/d2go/pkg/data/stat"
)

func TestPlayerUnitHPPercentAvoidsZeroDenominator(t *testing.T) {
	pu := PlayerUnit{
		Stats: stat.Stats{
			{ID: stat.Life, Value: 0},
			{ID: stat.MaxLife, Value: 0},
		},
	}

	if got := pu.HPPercent(); got != 0 {
		t.Fatalf("HPPercent() = %d; want 0", got)
	}
}

func TestPlayerUnitMPPercentAvoidsZeroDenominator(t *testing.T) {
	pu := PlayerUnit{
		Stats: stat.Stats{
			{ID: stat.Mana, Value: 0},
			{ID: stat.MaxMana, Value: 0},
		},
	}

	if got := pu.MPPercent(); got != 0 {
		t.Fatalf("MPPercent() = %d; want 0", got)
	}
}
