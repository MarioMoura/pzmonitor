// Package playersdb reads character data from a Project Zomboid server's
// players.db (Saves/Multiplayer/<server>/players.db).
//
// The networkPlayers.data blob is the serialized IsoPlayer. The layout below
// follows IsoMovingObject.save -> IsoGameCharacter.save -> IsoPlayer.save in
// Build 42 (world version 249). Only the fields needed for metrics are
// decoded; everything else is skipped using the counts and length prefixes
// the format provides.
package playersdb

import (
	"encoding/binary"
	"fmt"
	"math"
)

// StatNames is the order of CharacterStat.ORDERED_STATS.
var StatNames = []string{
	"anger", "boredom", "discomfort", "endurance", "fatigue", "fitness",
	"food_sickness", "hunger", "idleness", "intoxication", "morale",
	"nicotine_withdrawal", "pain", "panic", "poison", "sanity", "sickness",
	"stress", "temperature", "thirst", "unhappiness", "wetness",
	"zombie_fever", "zombie_infection",
}

const bodyParts = 17

// Character holds the decoded fields of one saved character.
type Character struct {
	Forename, Surname, Profession string
	Female                        bool
	X, Y, Z                       float64
	Items                         int
	Stats                         map[string]float64
	BodyPartHealth                []float64
	Traits                        []string
	PerkLevels                    map[string]int
	HoursSurvived                 float64
	ZombieKills, SurvivorKills    int
}

// Health returns the average body part health (0-100).
func (c *Character) Health() float64 {
	var sum float64
	for _, h := range c.BodyPartHealth {
		sum += h
	}
	return sum / float64(len(c.BodyPartHealth))
}

type reader struct {
	b   []byte
	p   int
	err error
}

func (r *reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if r.p+n > len(r.b) {
		r.err = fmt.Errorf("truncated blob at offset %d (need %d of %d)", r.p, n, len(r.b))
		return false
	}
	return true
}

func (r *reader) u8() byte {
	if !r.need(1) {
		return 0
	}
	v := r.b[r.p]
	r.p++
	return v
}

func (r *reader) i16() int {
	if !r.need(2) {
		return 0
	}
	v := int(int16(binary.BigEndian.Uint16(r.b[r.p:])))
	r.p += 2
	return v
}

func (r *reader) i32() int {
	if !r.need(4) {
		return 0
	}
	v := int(int32(binary.BigEndian.Uint32(r.b[r.p:])))
	r.p += 4
	return v
}

func (r *reader) f32() float64 {
	if !r.need(4) {
		return 0
	}
	v := math.Float32frombits(binary.BigEndian.Uint32(r.b[r.p:]))
	r.p += 4
	return float64(v)
}

func (r *reader) f64() float64 {
	if !r.need(8) {
		return 0
	}
	v := math.Float64frombits(binary.BigEndian.Uint64(r.b[r.p:]))
	r.p += 8
	return v
}

func (r *reader) skip(n int) {
	if n < 0 {
		r.err = fmt.Errorf("negative length %d at offset %d", n, r.p)
		return
	}
	if r.need(n) {
		r.p += n
	}
}

// str reads GameWindow.WriteString: int16 length + UTF-8 bytes.
func (r *reader) str() string {
	n := r.i16()
	if n < 0 {
		r.err = fmt.Errorf("negative string length at offset %d", r.p)
		return ""
	}
	if !r.need(n) {
		return ""
	}
	v := string(r.b[r.p : r.p+n])
	r.p += n
	return v
}

// count reads an int32 used as a loop bound and guards against garbage.
func (r *reader) count() int {
	n := r.i32()
	if n < 0 || n > 1<<20 {
		r.err = fmt.Errorf("implausible count %d at offset %d", n, r.p)
		return 0
	}
	return n
}

func (r *reader) luaValue(t byte) {
	switch t {
	case 0:
		r.str()
	case 1:
		r.f64()
	case 3:
		r.u8()
	case 2:
		r.luaTable()
	default:
		r.err = fmt.Errorf("invalid lua table type %d at offset %d", t, r.p)
	}
}

func (r *reader) luaTable() {
	n := r.count()
	for i := 0; i < n && r.err == nil; i++ {
		r.luaValue(r.u8())
		r.luaValue(r.u8())
	}
}

func (r *reader) itemVisual() {
	f := r.u8()
	r.str()
	r.str()
	r.str()
	if f&1 != 0 {
		r.skip(3)
	}
	if f&2 != 0 {
		r.skip(1)
	}
	if f&4 != 0 {
		r.skip(1)
	}
	if f&8 != 0 {
		r.f32()
	}
	if f&0x10 != 0 {
		r.str()
	}
	for i := 0; i < 6; i++ { // blood, dirt, holes, basic/denim/leather patches
		r.skip(int(r.u8()))
	}
}

func (r *reader) humanVisual() {
	f := r.u8()
	if f&4 != 0 {
		r.skip(3)
	}
	if f&2 != 0 {
		r.skip(3)
	}
	if f&8 != 0 {
		r.skip(3)
	}
	r.skip(3) // bodyHair, skinTexture, zombieRotStage
	if f&0x40 != 0 {
		r.str()
	}
	if f&0x10 != 0 {
		r.str()
	}
	if f&0x20 != 0 {
		r.str()
	}
	for i := 0; i < 3; i++ { // blood, dirt, holes
		r.skip(int(r.u8()))
	}
	n := int(r.u8())
	for i := 0; i < n && r.err == nil; i++ {
		r.itemVisual()
	}
	r.str() // nonAttachedHair
	f2 := r.u8()
	if f2&4 != 0 {
		r.skip(3)
	}
	if f2&2 != 0 {
		r.skip(3)
	}
}

func (r *reader) inventory() int {
	r.str() // container type
	r.u8()  // explored
	n := r.i16()
	for i := 0; i < n && r.err == nil; i++ {
		identical := r.i32()
		r.skip(r.i32()) // saveWithSize
		if identical > 1 {
			r.skip(4 * (identical - 1))
		}
	}
	r.u8()  // looted
	r.i32() // capacity
	return n
}

func (r *reader) bodyDamage() []float64 {
	health := make([]float64, 0, bodyParts)
	for i := 0; i < bodyParts && r.err == nil; i++ {
		r.skip(3)
		bandaged := r.u8()
		r.skip(4)
		health = append(health, r.f32())
		if bandaged != 0 {
			r.f32()
		}
		if r.u8() != 0 {
			r.f32() // wound infection level
		}
		r.skip(7 * 4)
		r.skip(3)
		r.f32()
		r.skip(2)
		r.f32()
		if r.u8() != 0 {
			r.f32() // splint factor
		}
		r.u8()
		r.f32()
		r.u8()
		r.f32()
		r.str()
		r.str()
		r.skip(6 * 4)
	}
	// saveMainFields
	r.f32()
	r.u8()
	r.f32()
	r.i32()
	r.u8()
	r.skip(6 * 4)
	if r.u8() != 0 { // thermoregulator
		r.skip(9 * 4)
		n := r.count()
		r.skip(n * 10 * 4)
	}
	return health
}

func (r *reader) xp(c *Character) {
	n := r.count()
	c.Traits = make([]string, 0, n)
	for i := 0; i < n && r.err == nil; i++ {
		c.Traits = append(c.Traits, r.str())
	}
	r.f32() // totalXp
	r.i32() // level
	r.i32() // lastlevel
	n = r.count()
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
		r.f32()
	}
	n = r.count()
	c.PerkLevels = make(map[string]int, n)
	for i := 0; i < n && r.err == nil; i++ {
		perk := r.str()
		c.PerkLevels[perk] = r.i32()
	}
	n = r.count()
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
		r.f32()
		r.skip(2)
	}
}

// Parse decodes a networkPlayers.data blob.
func Parse(blob []byte) (*Character, error) {
	r := &reader{b: blob}
	c := &Character{}

	// IsoMovingObject
	r.u8() // serialize flag
	r.u8() // class id
	r.f32()
	r.f32() // offsets
	c.X, c.Y, c.Z = r.f32(), r.f32(), r.f32()
	r.i32() // direction
	if r.u8() != 0 {
		r.luaTable() // modData
	}

	// IsoGameCharacter
	if r.u8() != 0 { // descriptor
		r.i32()
		c.Forename = r.str()
		c.Surname = r.str()
		r.str() // torso
		c.Female = r.i32() == 1
		c.Profession = r.str()
		if r.i32() != 0 {
			n := r.count()
			for i := 0; i < n && r.err == nil; i++ {
				r.str()
			}
		}
		n := r.count()
		for i := 0; i < n && r.err == nil; i++ {
			r.str()
			r.i32()
		}
		r.str()
		r.f32()
		r.i32()
	}
	r.humanVisual()
	c.Items = r.inventory()
	r.u8()  // asleep
	r.f32() // forceWakeUpTime
	c.Stats = make(map[string]float64, len(StatNames))
	for _, name := range StatNames {
		c.Stats[name] = r.f32()
	}
	c.BodyPartHealth = r.bodyDamage()
	r.xp(c)
	r.i32()
	r.i32() // hand item indexes
	r.u8()  // onFire
	r.skip(8 * 4)
	n := r.count() // readBooks
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
		r.i32()
	}
	r.f32()
	n = r.count() // knownRecipes
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
	}
	r.i32()
	r.skip(3 * 4)
	r.skip(15)    // cheat flags
	n = r.count() // readLiterature
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
		r.i32()
	}
	n = r.count() // readPrintMedia
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
	}
	r.skip(8)         // lastAnimalPet
	r.skip(r.count()) // cheats

	// IsoPlayer
	c.HoursSurvived = r.f64()
	c.ZombieKills = r.i32()
	n = int(r.u8()) // worn items
	for i := 0; i < n && r.err == nil; i++ {
		r.str()
		r.i16()
	}
	r.i16()
	r.i16()
	c.SurvivorKills = r.i32()

	if r.err != nil {
		return nil, r.err
	}
	return c, nil
}
