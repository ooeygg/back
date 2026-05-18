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
	unitTableOffset := uintptr(0x1EB7460)

	// UI
	uiOffsetPtr := uintptr(0x1EC715A)

	// Hover
	hoverOffset := uintptr(0x1E0B110)

	// Expansion
	expOffset := uintptr(0x1E0A578)

	// Party members offset
	rosterOffset := uintptr(0x1ECD778)

	// PanelManagerContainer
	panelManagerContainerOffset := uintptr(0x1E21ED0)

	// WidgetStates
	WidgetStatesOffset := uintptr(0x1EEF790)

	// Waypoints
	WaypointTableOffset := uintptr(0x1D694D0)

	// FPS
	fpsOffset := uintptr(0x1D694A4)

	// KeyBindings
	keyBindingsOffset := uintptr(0x19E1434)

	// KeyBindings Skills
	keyBindingsSkillsOffset := uintptr(0x1E0B220)

	// QuestInfo
	questInfoOffset := uintptr(0x1ED3DE8)

	// Terror Zones
	tzOffset := uintptr(0x25B1B80)

	// Ping
	pingOffset := uintptr(0x1E0A578)

	// LegacyGraphics
	legacyGfxOffset := uintptr(0x1ED4056)

	// CharData
	charDataOffset := uintptr(0x1E0E708)

	// Selected Char Name
	selectedCharNameOffset := uintptr(0x1D60295)

	// Last Game Name
	lastGameNameOffset := uintptr(0x25FA4E0)

	// Last Game Password
	lastGamePasswordOffset := uintptr(0x25FA538)

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
