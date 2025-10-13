package tiled

import (
	"encoding/xml"

	"github.com/adm87/enum"
)

// ======================================================
// Tmx - Tiled Map XML
// ======================================================

type Tmx struct {
	Width      int32 `xml:"width,attr"`
	Height     int32 `xml:"height,attr"`
	TileHeight int32 `xml:"tileheight,attr"`
	TileWidth  int32 `xml:"tilewidth,attr"`

	Flags       MapFlag     `xml:"-"`
	Orientation Orientation `xml:"-"`
	RenderOrder RenderOrder `xml:"-"`

	NextLayerID  int32 `xml:"nextlayerid,attr"`
	NextObjectID int32 `xml:"nextobjectid,attr"`

	Tilesets     []Tileset     `xml:"tileset,omitempty"`
	Layers       []Layer       `xml:"layer,omitempty"`
	ObjectGroups []ObjectGroup `xml:"objectgroup,omitempty"`

	Properties []Property `xml:"properties>property,omitempty"`
}

func (t *Tmx) IsInfinite() bool {
	return t.Flags&MapFlagInfinite != 0
}

func (t *Tmx) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "infinite":
			if attr.Value == "1" {
				t.Flags |= MapFlagInfinite
			}
		case "orientation":
			val, err := enum.UnmarshalEnum[Orientation](attr.Value)
			if err != nil {
				return err
			}
			t.Orientation = val
		case "renderorder":
			val, err := enum.UnmarshalEnum[RenderOrder](attr.Value)
			if err != nil {
				return err
			}
			t.RenderOrder = val
		}
	}

	type tmxAlias Tmx
	aux := (*tmxAlias)(t)

	return d.DecodeElement(aux, &start)
}

// ======================================================
// Tsx - Tiled Tileset XML
// ======================================================

type Tsx struct {
	TileWidth  int32 `xml:"tilewidth,attr"`
	TileHeight int32 `xml:"tileheight,attr"`
	TileCount  int32 `xml:"tilecount,attr"`
	Columns    int32 `xml:"columns,attr"`

	Image      Image  `xml:"image,omitempty"`
	TileOffset Offset `xml:"tileoffset,omitempty"`

	ObjectAlignment ObjectAlignment `xml:"-"`

	Properties []Property `xml:"properties>property,omitempty"`
}

func (t *Tsx) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "objectalignment":
			val, err := enum.UnmarshalEnum[ObjectAlignment](attr.Value)
			if err != nil {
				return err
			}
			t.ObjectAlignment = val
		}
	}

	type tsxAlias Tsx
	aux := (*tsxAlias)(t)

	return d.DecodeElement(aux, &start)
}

// ======================================================
// Data
// ======================================================

type Data struct {
	Encoding    Encoding    `xml:"-"`
	Compression Compression `xml:"-"`

	Chunks []Chunk `xml:"chunk,omitempty"`

	Content string `xml:",chardata"`
}

func (dt *Data) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "encoding":
			val, err := enum.UnmarshalEnum[Encoding](attr.Value)
			if err != nil {
				return err
			}
			dt.Encoding = val
		case "compression":
			val, err := enum.UnmarshalEnum[Compression](attr.Value)
			if err != nil {
				return err
			}
			dt.Compression = val
		}
	}

	type dataAlias Data
	aux := (*dataAlias)(dt)

	return d.DecodeElement(aux, &start)
}

// ======================================================
// ObjectGroup
// ======================================================

type ObjectGroup struct {
	Flags     LayerFlag `xml:"-"`
	DrawOrder DrawOrder `xml:"-"`

	ID   int32  `xml:"id,attr"`
	Name string `xml:"name,attr"`

	Objects    []Object   `xml:"object,omitempty"`
	Properties []Property `xml:"properties>property,omitempty"`
}

func (og *ObjectGroup) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	og.Flags |= LayerFlagVisible

	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "visible":
			if attr.Value == "0" {
				og.Flags &^= LayerFlagVisible
			} else {
				og.Flags |= LayerFlagVisible
			}
		case "locked":
			if attr.Value != "" {
				og.Flags |= LayerFlagLocked
			} else {
				og.Flags &^= LayerFlagLocked
			}
		case "draworder":
			val, err := enum.UnmarshalEnum[DrawOrder](attr.Value)
			if err != nil {
				return err
			}
			og.DrawOrder = val
		}
	}

	type objectgroupAlias ObjectGroup
	aux := (*objectgroupAlias)(og)

	return d.DecodeElement(aux, &start)
}

// ======================================================
// Object
// ======================================================

type Object struct {
	X        float32 `xml:"x,attr"`
	Y        float32 `xml:"y,attr"`
	Width    float32 `xml:"width,attr,omitempty"`
	Height   float32 `xml:"height,attr,omitempty"`
	Rotation float32 `xml:"rotation,attr,omitempty"`

	Flags ObjectFlag `xml:"-"`

	ID       int32  `xml:"id,attr"`
	GID      uint32 `xml:"gid,attr,omitempty"`
	Name     string `xml:"name,attr,omitempty"`
	Template string `xml:"template,attr,omitempty"`

	Properties []Property `xml:"properties>property,omitempty"`
}

func (o *Object) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	o.Flags |= ObjectFlagVisible

	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "visible":
			if attr.Value == "0" {
				o.Flags &^= ObjectFlagVisible
			} else {
				o.Flags |= ObjectFlagVisible
			}
		case "template":
			if attr.Value != "" {
				o.Flags |= ObjectFlagTemplate
			} else {
				o.Flags &^= ObjectFlagTemplate
			}
		}
	}

	type objectAlias Object
	aux := (*objectAlias)(o)

	return d.DecodeElement(aux, &start)
}

func (o *Object) IsVisible() bool {
	return o.Flags&ObjectFlagVisible != 0
}

func (o *Object) IsTemplate() bool {
	return o.Flags&ObjectFlagTemplate != 0
}

// ======================================================
// Layer
// ======================================================

type Layer struct {
	Width  int32 `xml:"width,attr"`
	Height int32 `xml:"height,attr"`

	Flags LayerFlag `xml:"-"`

	Data Data `xml:"data,omitempty"`

	ID   int32  `xml:"id,attr"`
	Name string `xml:"name,attr"`

	Properties []Property `xml:"properties>property,omitempty"`
}

func (l *Layer) IsLocked() bool {
	return l.Flags&LayerFlagLocked != 0
}

func (l *Layer) IsVisible() bool {
	return l.Flags&LayerFlagVisible != 0
}

func (l *Layer) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	l.Flags |= LayerFlagVisible

	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "visible":
			if attr.Value == "0" {
				l.Flags &^= LayerFlagVisible
			}
		case "locked":
			if attr.Value != "" {
				l.Flags |= LayerFlagLocked
			} else {
				l.Flags &^= LayerFlagLocked
			}
		}
	}

	type layerAlias Layer
	aux := (*layerAlias)(l)

	return d.DecodeElement(aux, &start)
}

// ======================================================
// Tx - Tiled Template XML
// ======================================================

type Tx struct {
	Tileset Tileset `xml:"tileset,omitempty"`
	Objects Object  `xml:"object,omitempty"`
}

// ======================================================
// Image
// ======================================================

type Image struct {
	Width  int32 `xml:"width,attr,omitempty"`
	Height int32 `xml:"height,attr,omitempty"`

	Source string `xml:"source,attr,omitempty"`
}

// ======================================================
// Offset
// ======================================================

type Offset struct {
	X int32 `xml:"x,attr,omitempty"`
	Y int32 `xml:"y,attr,omitempty"`
}

// ======================================================
// Tileset
// ======================================================

type Tileset struct {
	FirstGID uint32 `xml:"firstgid,attr,omitempty"`
	Source   string `xml:"source,attr,omitempty"`
}

// ======================================================
// Chunk
// ======================================================

type Chunk struct {
	X      int32    `xml:"x,attr"`
	Y      int32    `xml:"y,attr"`
	Width  int32    `xml:"width,attr"`
	Height int32    `xml:"height,attr"`
	Tiles  []uint32 `xml:"-"`

	Content string `xml:",chardata"`
}

// ======================================================
// Property
// ======================================================

type Property struct {
	Value        string `xml:"value,attr"`
	PropertyType string `xml:"propertytype,attr,omitempty"`

	Name string `xml:"name,attr"`

	Properties []Property `xml:"properties>property,omitempty"`
}
