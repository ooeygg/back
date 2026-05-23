package memory

import (
	"math"
	"sort"

	"github.com/ooeygg/remas/back/d2go/pkg/data"
	"github.com/ooeygg/remas/back/d2go/pkg/data/mode"
	"github.com/ooeygg/remas/back/d2go/pkg/data/npc"
	"github.com/ooeygg/remas/back/d2go/pkg/data/stat"
)

type OwnMercenary struct {
	Found       bool
	ClassID     npc.ID
	UnitID      data.UnitID
	OwnerUnitID data.UnitID
	OwnerName   string
	Position    data.Position
	Mode        mode.NpcMode
	IsCorpse    bool
	HPPercent   int
}

func (m OwnMercenary) Alive() bool {
	return m.Found && !m.IsCorpse && m.HPPercent > 0 && m.Mode != mode.NpcDeath && m.Mode != mode.NpcDead
}

func (gd *GameReader) OwnMercenary() (OwnMercenary, bool) {
	if gd.offset.UnitTable == 0 {
		gd.offset = calculateOffsets(gd.Process)
	}

	ownerIDs := gd.mainPlayerOwnerIDs()
	if len(ownerIDs) == 0 {
		return OwnMercenary{}, false
	}

	// Capture the main player name for ownership verification at the caller level.
	mainPlayerName := gd.GetRawPlayerUnits().GetMainPlayer().Name

	baseAddr := gd.Process.moduleBaseAddressPtr + gd.offset.UnitTable + 1024
	unitTableBuffer := gd.Process.ReadBytesFromMemory(baseAddr, 128*8)
	candidates := make([]OwnMercenary, 0)

	for bucket := 0; bucket < 128; bucket++ {
		monsterOffset := uint(8 * bucket)
		monsterUnitPtr := uintptr(ReadUIntFromBuffer(unitTableBuffer, monsterOffset, Uint64))
		for depth := 0; depth < 64 && monsterUnitPtr != 0; depth++ {
			monsterDataBuffer := gd.Process.ReadBytesFromMemory(monsterUnitPtr, 0x1B0)
			classID := npc.ID(ReadUIntFromBuffer(monsterDataBuffer, 0x04, Uint32))
			if !isMercenaryClassID(classID) {
				monsterUnitPtr = uintptr(gd.Process.ReadUInt(monsterUnitPtr+0x158, Uint64))
				continue
			}

			unitDataPtr := uintptr(ReadUIntFromBuffer(monsterDataBuffer, 0x10, Uint64))
			unitOwnerID := uint32(ReadUIntFromBuffer(monsterDataBuffer, 0x08, Uint32))
			unitDataOwnerID := uint32(0)
			if unitDataPtr != 0 {
				unitDataOwnerID = uint32(gd.Process.ReadUInt(unitDataPtr+0x54, Uint32))
			}
			ownerID, ownerMatched := matchOwnerUnitID(ownerIDs, unitOwnerID, unitDataOwnerID)
			if !ownerMatched {
				monsterUnitPtr = uintptr(gd.Process.ReadUInt(monsterUnitPtr+0x158, Uint64))
				continue
			}

			pathPtr := uintptr(ReadUIntFromBuffer(monsterDataBuffer, 0x38, Uint64))
			xPos := int(gd.Process.ReadUInt(pathPtr+0x02, Uint16))
			yPos := int(gd.Process.ReadUInt(pathPtr+0x06, Uint16))
			statsListExPtr := uintptr(ReadUIntFromBuffer(monsterDataBuffer, 0x88, Uint64))
			statPtr := uintptr(gd.Process.ReadUInt(statsListExPtr+0x30, Uint64))
			statCount := gd.Process.ReadUInt(statsListExPtr+0x38, Uint64)
			stats := gd.getMonsterStats(statCount, statPtr)

			candidates = append(candidates, OwnMercenary{
				Found:       true,
				ClassID:     classID,
				UnitID:      data.UnitID(unitOwnerID),
				OwnerUnitID: data.UnitID(ownerID),
				OwnerName:   mainPlayerName,
				Position: data.Position{
					X: xPos,
					Y: yPos,
				},
				Mode:      mode.NpcMode(ReadUIntFromBuffer(monsterDataBuffer, 0x0C, Uint32)),
				IsCorpse:  ReadUIntFromBuffer(monsterDataBuffer, 0x1AE, Uint8) != 0,
				HPPercent: mercenaryHPPercent(stats),
			})

			monsterUnitPtr = uintptr(gd.Process.ReadUInt(monsterUnitPtr+0x158, Uint64))
		}
	}

	if len(candidates) == 0 {
		return OwnMercenary{}, false
	}

	// Filter candidates to only the mercenary owned by the main (local) player.
	// In a Wine/Proton party multiple players may have IsMainPlayer=true, so the
	// raw candidates list can contain teammates' mercenaries. We filter by the
	// owner name captured from the main player unit before sorting.
	ownCandidates := make([]OwnMercenary, 0, len(candidates))
	for _, c := range candidates {
		if c.OwnerName == mainPlayerName {
			ownCandidates = append(ownCandidates, c)
		}
	}
	if len(ownCandidates) == 0 {
		return OwnMercenary{}, false
	}

	sort.SliceStable(ownCandidates, func(i, j int) bool {
		if ownCandidates[i].Alive() != ownCandidates[j].Alive() {
			return ownCandidates[i].Alive()
		}
		if ownCandidates[i].ClassID == npc.Guard && ownCandidates[j].ClassID != npc.Guard {
			return true
		}
		if ownCandidates[i].ClassID != npc.Guard && ownCandidates[j].ClassID == npc.Guard {
			return false
		}
		return ownCandidates[i].HPPercent > ownCandidates[j].HPPercent
	})

	return ownCandidates[0], true
}

func (gd *GameReader) mainPlayerOwnerIDs() map[uint32]struct{} {
	rawPlayerUnits := gd.GetRawPlayerUnits()
	ownerIDs := make(map[uint32]struct{})
	for _, player := range rawPlayerUnits {
		if !player.IsMainPlayer || player.UnitID <= 0 {
			continue
		}
		if player.Area > 0 && player.Position.X > 0 && player.Position.Y > 0 {
			ownerIDs[uint32(player.UnitID)] = struct{}{}
		}
	}
	if len(ownerIDs) > 0 {
		return ownerIDs
	}
	for _, player := range rawPlayerUnits {
		if player.IsMainPlayer && player.UnitID > 0 {
			ownerIDs[uint32(player.UnitID)] = struct{}{}
		}
	}
	return ownerIDs
}

func isMercenaryClassID(classID npc.ID) bool {
	switch classID {
	case npc.Guard, npc.Act5Hireling1Hand, npc.Act5Hireling2Hand, npc.IronWolf, npc.Rogue2:
		return true
	default:
		return false
	}
}

func matchOwnerUnitID(ownerIDs map[uint32]struct{}, unitOwnerID, unitDataOwnerID uint32) (uint32, bool) {
	if _, ok := ownerIDs[unitDataOwnerID]; ok {
		return unitDataOwnerID, true
	}
	if _, ok := ownerIDs[unitOwnerID]; ok {
		return unitOwnerID, true
	}
	return 0, false
}

func mercenaryHPPercent(stats map[stat.ID]int) int {
	maxLife := stats[stat.MaxLife] >> 8
	if maxLife <= 0 {
		return 0
	}

	life := float64(stats[stat.Life] >> 8)
	if stats[stat.Life] <= 32768 {
		life = float64(stats[stat.Life]) / 32768.0 * float64(maxLife)
	}

	percentage := (life / float64(maxLife)) * 100
	if math.IsNaN(percentage) || math.IsInf(percentage, 0) {
		return 0
	}
	if percentage < 0 {
		return 0
	}
	if percentage > 100 {
		return 100
	}
	return int(percentage)
}
