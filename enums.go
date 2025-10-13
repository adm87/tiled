package tiled

// ======================================================
// DrawOrder
// ======================================================

type DrawOrder uint8

const (
	DrawOrderIndex DrawOrder = iota
)

func (do DrawOrder) String() string {
	switch do {
	case DrawOrderIndex:
		return "index"
	default:
		return "unknown"
	}
}

func (do DrawOrder) IsValid() bool {
	return do >= DrawOrderIndex && do <= DrawOrderIndex
}

// ======================================================
// Compression
// ======================================================

type Compression uint8

const (
	CompressionNone Compression = iota
	CompressionGzip
	CompressionZlib
	CompressionZstd
)

func (c Compression) String() string {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionGzip:
		return "gzip"
	case CompressionZlib:
		return "zlib"
	case CompressionZstd:
		return "zstd"
	default:
		return "unknown"
	}
}

func (c Compression) IsValid() bool {
	return c >= CompressionNone && c <= CompressionZstd
}

// ======================================================
// Encoding
// ======================================================

type Encoding uint8

const (
	EncodingCSV Encoding = iota
	EncodingBase64
)

func (e Encoding) String() string {
	switch e {
	case EncodingCSV:
		return "csv"
	case EncodingBase64:
		return "base64"
	default:
		return "unknown"
	}
}

func (e Encoding) IsValid() bool {
	return e >= EncodingCSV && e <= EncodingBase64
}

// ======================================================
// ObjectAlignment
// ======================================================

type ObjectAlignment uint8

const (
	ObjectAlignmentUnspecified ObjectAlignment = iota
	ObjectAlignmentTopLeft
	ObjectAlignmentTop
	ObjectAlignmentTopRight
	ObjectAlignmentLeft
	ObjectAlignmentCenter
	ObjectAlignmentRight
	ObjectAlignmentBottomLeft
	ObjectAlignmentBottom
	ObjectAlignmentBottomRight
)

func (oa ObjectAlignment) String() string {
	switch oa {
	case ObjectAlignmentUnspecified:
		return "unspecified"
	case ObjectAlignmentTopLeft:
		return "topleft"
	case ObjectAlignmentTop:
		return "top"
	case ObjectAlignmentTopRight:
		return "topright"
	case ObjectAlignmentLeft:
		return "left"
	case ObjectAlignmentCenter:
		return "center"
	case ObjectAlignmentRight:
		return "right"
	case ObjectAlignmentBottomLeft:
		return "bottomleft"
	case ObjectAlignmentBottom:
		return "bottom"
	case ObjectAlignmentBottomRight:
		return "bottomright"
	default:
		return "unknown"
	}
}

func (oa ObjectAlignment) IsValid() bool {
	return oa >= ObjectAlignmentUnspecified && oa <= ObjectAlignmentBottomRight
}

// ======================================================
// Orientation
// ======================================================

type Orientation uint8

const (
	OrientationOrthogonal Orientation = iota
	OrientationIsometric
	OrientationStaggered
	OrientationHexagonal
)

func (o Orientation) String() string {
	switch o {
	case OrientationOrthogonal:
		return "orthogonal"
	case OrientationIsometric:
		return "isometric"
	case OrientationStaggered:
		return "staggered"
	case OrientationHexagonal:
		return "hexagonal"
	default:
		return "unknown"
	}
}

func (o Orientation) IsValid() bool {
	return o >= OrientationOrthogonal && o <= OrientationHexagonal
}

// ======================================================
// RenderOrder
// ======================================================

type RenderOrder uint8

const (
	RenderOrderRightDown RenderOrder = iota
	RenderOrderRightUp
	RenderOrderLeftDown
	RenderOrderLeftUp
)

func (ro RenderOrder) String() string {
	switch ro {
	case RenderOrderRightDown:
		return "right-down"
	case RenderOrderRightUp:
		return "right-up"
	case RenderOrderLeftDown:
		return "left-down"
	case RenderOrderLeftUp:
		return "left-up"
	default:
		return "unknown"
	}
}

func (ro RenderOrder) IsValid() bool {
	return ro >= RenderOrderRightDown && ro <= RenderOrderLeftUp
}
