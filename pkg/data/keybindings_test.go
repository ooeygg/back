package data

import (
	"testing"

	"github.com/ooeygg/remas/back/d2go/pkg/data/skill"
)

func TestKeyBindingForSkillTownPortalAlias(t *testing.T) {
	expected := KeyBinding{
		Key1: [2]byte{0x70, 0x00},
		Key2: [2]byte{0x00, 0x00},
	}

	kb := KeyBindings{
		Skills: [16]SkillBinding{
			{
				SkillID:    skill.ScrollOfTownPortal,
				KeyBinding: expected,
			},
		},
	}

	got, ok := kb.KeyBindingForSkill(skill.TomeOfTownPortal)
	if !ok {
		t.Fatalf("KeyBindingForSkill(%v) returned ok=false; want true", skill.TomeOfTownPortal)
	}
	if got != expected {
		t.Fatalf("KeyBindingForSkill(%v) = %#v; want %#v", skill.TomeOfTownPortal, got, expected)
	}
}
