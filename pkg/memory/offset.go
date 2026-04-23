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
	unitTableOffset := uintptr(0x1EA73D0)

	// UI
	uiOffsetPtr := uintptr(0x1EB70CA)

	// Hover
	hoverOffset := uintptr(0x1DFB080)

	// Expansion
	expOffset := uintptr(0x1DFA4E8)

	// Party members offset
	rosterOffset := uintptr(0x1EBD6E8)

	// PanelManagerContainer
	panelManagerContainerOffset := uintptr(0x1E11E40)

	// WidgetStates
	WidgetStatesOffset := uintptr(0x1EDF700)

	// Waypoints
	WaypointTableOffset := uintptr(0x1D59440)

	// FPS
	fpsOffset := uintptr(0x1D59414)

	// KeyBindings
	keyBindingsOffset := uintptr(0x19D25B4)

	// KeyBindings Skills
	keyBindingsSkillsOffset := uintptr(0x1DFB190)

	// QuestInfo
	questInfoOffset := uintptr(0x1EC3D58)

	// Terror Zones
	tzOffset := uintptr(0x25B1B80)

	// Ping
	pingOffset := uintptr(0x1DFA4E8)

	// LegacyGraphics
	legacyGfxOffset := uintptr(0x1EC3FC6)

	// CharData
	charDataOffset := uintptr(0x1DFE678)

	// Selected Char Name
	selectedCharNameOffset := uintptr(0x1D50215)

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
