package tiled

func LayerByName(tmx *Tmx, name string) *Layer {
	for i := range tmx.Layers {
		if tmx.Layers[i].Name == name {
			return &tmx.Layers[i]
		}
	}
	return nil
}

func ObjectGroupByName(tmx *Tmx, name string) *ObjectGroup {
	for i := range tmx.ObjectGroups {
		if tmx.ObjectGroups[i].Name == name {
			return &tmx.ObjectGroups[i]
		}
	}
	return nil
}

func TilesetByGID(tmx *Tmx, gid uint32) (*Tileset, uint32, int) {
	for i := len(tmx.Tilesets) - 1; i >= 0; i-- {
		if gid >= tmx.Tilesets[i].FirstGID {
			return &tmx.Tilesets[i], gid - tmx.Tilesets[i].FirstGID, i
		}
	}
	return nil, 0, -1
}

func PropertyByName(props []Property, name string) *Property {
	for i := range props {
		if props[i].Name == name {
			return &props[i]
		}
	}
	return nil
}

func PropertyByType(props []Property, propertyType string) *Property {
	for i := range props {
		if props[i].PropertyType == propertyType {
			return &props[i]
		}
	}
	return nil
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
