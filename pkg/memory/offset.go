package memory

type Offset struct {
	GameData                    uintptr
	UnitTable                   uintptr
	UI                          uintptr
	Hover                       uintptr
	Expansion                   uintptr
	RosterOffset                uintptr
	PanelManagerContainerOffset uintptr
	WidgetStatesOffset          uintptr
	WaypointTableOffset         uintptr
	FPS                         uintptr
	KeyBindingsOffset           uintptr
	KeyBindingsSkillsOffset     uintptr
	QuestInfo                   uintptr
	TZ                          uintptr
	Quests                      uintptr
	Ping                        uintptr
	LegacyGraphics              uintptr
	CharData                    uintptr
	SelectedCharName            uintptr
	LastGameName                uintptr
	LastGamePassword            uintptr
}

func calculateOffsets(_ *Process) Offset {
	// UnitTable
	unitTableOffset := uintptr(0x1EB9430)

	// UI
	uiOffsetPtr := uintptr(0x1EC912A)

	// Hover
	hoverOffset := uintptr(0x1E0D0A0)

	// Expansion
	expOffset := uintptr(0x1E0C508)

	// Party members offset
	rosterOffset := uintptr(0x1ECF748)

	// PanelManagerContainer
	panelManagerContainerOffset := uintptr(0x1E23E60)

	// WidgetStates
	WidgetStatesOffset := uintptr(0x1EF1760)

	// Waypoints
	WaypointTableOffset := uintptr(0x1D6B458)

	// FPS
	fpsOffset := uintptr(0x1D6B42C)

	// KeyBindings
	keyBindingsOffset := uintptr(0x19E45F4)

	// KeyBindings Skills
	keyBindingsSkillsOffset := uintptr(0x1E0D1B0)

	// QuestInfo
	questInfoOffset := uintptr(0x1ED5DB8)

	// Terror Zones
	tzOffset := uintptr(0x25B5578)

	// Ping
	pingOffset := uintptr(0x1E0C508)

	// LegacyGraphics
	legacyGfxOffset := uintptr(0x1ED6026)

	// CharData
	charDataOffset := uintptr(0x1E10658)

	// Selected Char Name
	selectedCharNameOffset := uintptr(0x1D62215)

	// Last Game Name
	lastGameNameOffset := uintptr(0x260C4B0)

	// Last Game Password
	lastGamePasswordOffset := uintptr(0x260C508)

	return Offset{
		UnitTable:                   unitTableOffset,
		UI:                          uiOffsetPtr,
		Hover:                       hoverOffset,
		Expansion:                   expOffset,
		RosterOffset:                rosterOffset,
		PanelManagerContainerOffset: panelManagerContainerOffset,
		WidgetStatesOffset:          WidgetStatesOffset,
		WaypointTableOffset:         WaypointTableOffset,
		FPS:                         fpsOffset,
		KeyBindingsOffset:           keyBindingsOffset,
		KeyBindingsSkillsOffset:     keyBindingsSkillsOffset,
		QuestInfo:                   questInfoOffset,
		TZ:                          tzOffset,
		Ping:                        pingOffset,
		LegacyGraphics:              legacyGfxOffset,
		CharData:                    charDataOffset,
		SelectedCharName:            selectedCharNameOffset,
		LastGameName:                lastGameNameOffset,
		LastGamePassword:            lastGamePasswordOffset,
	}
}
