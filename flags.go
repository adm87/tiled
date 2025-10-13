package tiled

import "strings"

// ======================================================
// TmxFlag
// ======================================================

type MapFlag uint8

const (
	MapFlagInfinite MapFlag = 1 << iota

	mapFlagMax = MapFlagInfinite
)

func (f MapFlag) String() string {
	var flags []string
	if f&MapFlagInfinite != 0 {
		flags = append(flags, "infinite")
	}
	if len(flags) == 0 {
		return "None"
	}
	return strings.Join(flags, "|")
}

func (f MapFlag) IsValid() bool {
	return f&^mapFlagMax == 0
}

// ======================================================
// LayerFlags
// ======================================================

type LayerFlag uint8

const (
	LayerFlagLocked LayerFlag = 1 << iota
	LayerFlagVisible

	layerFlagMax = LayerFlagLocked | LayerFlagVisible
)

func (lf LayerFlag) String() string {
	var flags []string
	if lf&LayerFlagLocked != 0 {
		flags = append(flags, "locked")
	}
	if lf&LayerFlagVisible != 0 {
		flags = append(flags, "visible")
	}
	if len(flags) == 0 {
		return "None"
	}
	return strings.Join(flags, "|")
}

func (lf LayerFlag) IsValid() bool {
	return lf&^layerFlagMax == 0
}

// ======================================================
// ObjectFlag
// ======================================================

type ObjectFlag uint8

const (
	ObjectFlagVisible ObjectFlag = 1 << iota
	ObjectFlagTemplate

	objectFlagMax = ObjectFlagVisible | ObjectFlagTemplate
)

func (of ObjectFlag) String() string {
	var flags []string
	if of&ObjectFlagVisible != 0 {
		flags = append(flags, "visible")
	}
	if of&ObjectFlagTemplate != 0 {
		flags = append(flags, "template")
	}
	if len(flags) == 0 {
		return "None"
	}
	return strings.Join(flags, "|")
}

func (of ObjectFlag) IsValid() bool {
	return of&^objectFlagMax == 0
}

// ======================================================
// FlipFlag
// ======================================================

type FlipFlag uint8

const (
	FlipHorizontal FlipFlag = 1 << iota
	FlipVertical
	FlipDiagonal
	FlipHex

	flipFlagMax = FlipHorizontal | FlipVertical | FlipDiagonal | FlipHex
)

func (ff FlipFlag) String() string {
	var flags []string
	if ff&FlipHorizontal != 0 {
		flags = append(flags, "horizontal")
	}
	if ff&FlipVertical != 0 {
		flags = append(flags, "vertical")
	}
	if ff&FlipDiagonal != 0 {
		flags = append(flags, "diagonal")
	}
	if ff&FlipHex != 0 {
		flags = append(flags, "hex")
	}
	if len(flags) == 0 {
		return "None"
	}
	return strings.Join(flags, "|")
}

func (ff FlipFlag) IsValid() bool {
	return ff&^flipFlagMax == 0
}

func (ff FlipFlag) Horizontal() bool {
	return ff&FlipHorizontal != 0
}

func (ff FlipFlag) Vertical() bool {
	return ff&FlipVertical != 0
}

func (ff FlipFlag) Diagonal() bool {
	return ff&FlipDiagonal != 0
}

func (ff FlipFlag) Hex() bool {
	return ff&FlipHex != 0
}
