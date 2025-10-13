package shared

import (
	"embed"
	"encoding/xml"
	"strings"

	"github.com/adm87/tiled"
)

const (
	TilemapPacked           = "tilemap_packed.png"
	TilemapCharactersPacked = "tilemap-characters_packed.png"
	TilemapExampleA         = "tilemap-example-a.tmx"
	TilemapExampleB         = "tilemap-example-b.tmx"
	TilesetCharacters       = "tileset-characters.tsx"
	TilesetTiles            = "tileset-tiles.tsx"
)

//go:embed assets
var assets embed.FS

func LoadTiledAsset[T tiled.Tmx | tiled.Tsx | tiled.Tx](filename string) (*T, error) {
	file, err := LoadAsset(filename)
	if err != nil {
		return nil, err
	}

	var t T
	if err := xml.Unmarshal(file, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func MustLoadTiledAsset[T tiled.Tmx | tiled.Tsx | tiled.Tx](filename string) *T {
	t, err := LoadTiledAsset[T](filename)
	if err != nil {
		panic(err)
	}
	return t
}

func LoadAsset(filename string) ([]byte, error) {
	if !strings.HasPrefix(filename, "assets/") {
		filename = "assets/" + filename
	}
	return assets.ReadFile(filename)
}

func MustLoadImageAsset(filename string) []byte {
	data, err := LoadAsset(filename)
	if err != nil {
		panic(err)
	}
	return data
}
