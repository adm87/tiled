package tiled

import (
	"errors"
	"math"
)

var (
	ErrNoTmxData       = errors.New("no Tmx data set")
	ErrInvalidTmxData  = errors.New("invalid Tmx data")
	ErrTilesetNotFound = errors.New("tileset not found")
	ErrTileNotFound    = errors.New("tile not found")
	ErrTilesetSource   = errors.New("tileset source is empty")
)

// TileData represents a single tile instance in the tilemap with its properties.
type TileData struct {
	X, Y     int32    // World position
	TileID   uint32   // Tile ID
	TsIdx    int      // Tileset index
	FlipFlag FlipFlag // Flip flags
}

// TileRegion defines a rectangular region in tile coordinates.
type TileRegion struct {
	MinX, MinY int32
	MaxX, MaxY int32
}

func (tr TileRegion) Equal(other TileRegion) bool {
	return tr.MinX == other.MinX && tr.MinY == other.MinY && tr.MaxX == other.MaxX && tr.MaxY == other.MaxY
}

// TileIterator iterates over layers of TileData in a tilemap.
// Each call to Next() returns the tiles for the next layer as a slice.
// If a layer is not visible, Next() returns an empty slice for that layer.
// When all layers have been iterated, Next() returns nil and false.
type TileIterator struct {
	tiles     []TileData
	positions []int
	index     int
}

// Next returns the next layer of tiles.
func (ti *TileIterator) Next() []TileData {
	if ti.index >= len(ti.positions)-1 {
		return nil
	}

	start := ti.positions[ti.index]
	end := ti.positions[ti.index+1]
	ti.index++

	return ti.tiles[start:end]
}

func (ti *TileIterator) HasNext() bool {
	return ti.index < len(ti.positions)-1
}

func (ti *TileIterator) Index() int {
	return ti.index
}

func (ti *TileIterator) Reset() {
	ti.index = 0
}

// TilemapContent represents the decoded tile data for a layer or chunk.
type TilemapContent []uint32

type TileKey uint64

func NewTileKey(x, y int32) TileKey {
	return TileKey(int64(x)<<32 | int64(uint32(y)))
}

// TilemapLayer represents a layer in the tilemap, which can be either a full layer or composed of chunks.
type TilemapLayer struct {
	Content TilemapContent
	Chunks  []TilemapContent

	Tiles map[TileKey]TileData // Indexed tiles for quick lookup
}

// Tilemap provides an API for operating on deserialized Tmx data.
type Tilemap struct {
	Tmx *Tmx

	tileRegion TileRegion // Last queried tile region
	iterator   TileIterator

	minX, minY int32 // Minimum tile coordinates boundary
	maxX, maxY int32 // Maximum tile coordinates boundary

	decodedLayers []TilemapLayer
}

func NewTilemap() *Tilemap {
	return &Tilemap{
		Tmx:           nil,
		tileRegion:    TileRegion{},
		iterator:      TileIterator{},
		decodedLayers: make([]TilemapLayer, 0),
	}
}

func NewTilemapWithTmx(tmx *Tmx) (*Tilemap, error) {
	tm := NewTilemap()
	if err := tm.SetTmx(tmx); err != nil {
		return nil, err
	}
	return tm, nil
}

func (tm *Tilemap) SetTmx(tmx *Tmx) error {
	if tmx == nil || len(tmx.Layers) == 0 {
		return ErrInvalidTmxData
	}

	tm.FlushCache()

	layers, err := decodeTilemapLayers(tmx)
	if err != nil {
		return err
	}

	minX, minY, maxX, maxY := calculateTileBounds(tmx)

	tm.Tmx = tmx
	tm.minX = minX
	tm.minY = minY
	tm.maxX = maxX
	tm.maxY = maxY
	tm.decodedLayers = layers
	return nil
}

func (tm *Tilemap) FlushCache() {
	tm.decodedLayers = tm.decodedLayers[:0]
	tm.iterator.tiles = tm.iterator.tiles[:0]
	tm.iterator.positions = tm.iterator.positions[:0]
	tm.iterator.index = 0
	tm.tileRegion.MinX = 0
	tm.tileRegion.MinY = 0
	tm.tileRegion.MaxX = 0
	tm.tileRegion.MaxY = 0
}

// Bounds returns the world coordinate bounds of the tilemap.
func (tm *Tilemap) Bounds() (minX, minY, maxX, maxY int32) {
	return tm.minX, tm.minY, tm.maxX, tm.maxY
}

func (tm *Tilemap) GetTileset(index int) (*Tileset, error) {
	if tm.Tmx == nil || len(tm.Tmx.Tilesets) == 0 {
		return nil, ErrNoTmxData
	}

	if index < 0 || index >= len(tm.Tmx.Tilesets) {
		return nil, ErrTilesetNotFound
	}

	ts := &tm.Tmx.Tilesets[index]
	if ts.Source == "" {
		return nil, ErrTilesetSource
	}

	return ts, nil
}

// GetTiles returns a tile iterator for the provided region.
//
// Panics if the tilemap has no Tmx data set.
func (tm *Tilemap) GetTiles(minX, minY, maxX, maxY int32) (TileIterator, error) {
	if tm.Tmx == nil || len(tm.decodedLayers) == 0 {
		return TileIterator{}, ErrNoTmxData
	}

	queryRegion := calculateQueryRegion(minX, minY, maxX, maxY, tm.Tmx.TileWidth, tm.Tmx.TileHeight)

	if queryRegion.Equal(tm.tileRegion) {
		return TileIterator{
			tiles:     tm.iterator.tiles,
			positions: tm.iterator.positions,
			index:     0,
		}, nil
	}

	tm.tileRegion = queryRegion

	tm.iterator.tiles = tm.iterator.tiles[:0]
	tm.iterator.positions = tm.iterator.positions[:0]
	tm.iterator.index = 0

	for i := range tm.decodedLayers {
		tm.iterator.positions = append(tm.iterator.positions, len(tm.iterator.tiles))

		if !tm.Tmx.Layers[i].IsVisible() {
			continue
		}

		for y := queryRegion.MinY; y < queryRegion.MaxY; y++ {
			for x := queryRegion.MinX; x < queryRegion.MaxX; x++ {
				if tile, found := getTileAt(tm.Tmx, &tm.decodedLayers[i], x, y, i); found {
					tm.iterator.tiles = append(tm.iterator.tiles, tile)
				}
			}
		}
	}

	tm.iterator.positions = append(tm.iterator.positions, len(tm.iterator.tiles))
	tm.iterator.Reset()

	return TileIterator{
		tiles:     tm.iterator.tiles,
		positions: tm.iterator.positions,
		index:     0,
	}, nil
}

func decodeTilemapLayers(tmx *Tmx) ([]TilemapLayer, error) {
	layers := make([]TilemapLayer, len(tmx.Layers))

	for i := range tmx.Layers {
		layers[i].Tiles = make(map[TileKey]TileData)

		if tmx.IsInfinite() {
			chunks, err := decodeTilemapChunks(&tmx.Layers[i])
			if err != nil {
				return nil, err
			}
			layers[i].Chunks = chunks
			continue
		}

		data, err := DecodeContent(tmx.Layers[i].Data.Content, tmx.Layers[i].Data.Encoding, tmx.Layers[i].Data.Compression)
		if err != nil {
			return nil, err
		}
		layers[i].Content = data
	}

	return layers, nil
}

func decodeTilemapChunks(layer *Layer) ([]TilemapContent, error) {
	chunks := make([]TilemapContent, len(layer.Data.Chunks))

	for i := range layer.Data.Chunks {
		data, err := DecodeContent(layer.Data.Chunks[i].Content, layer.Data.Encoding, layer.Data.Compression)
		if err != nil {
			return nil, err
		}
		chunks[i] = data
	}

	return chunks, nil
}

func getTileAt(tmx *Tmx, layer *TilemapLayer, x, y int32, layerIdx int) (TileData, bool) {
	if tmx.IsInfinite() {
		return getChunkTileAt(tmx, layer, x, y, layerIdx)
	}

	if x < 0 || x >= tmx.Width || y < 0 || y >= tmx.Height {
		return TileData{}, false
	}

	var zero TileData

	idx := NewTileKey(x, y)
	if tile, exists := layer.Tiles[idx]; exists {
		return tile, true
	}

	i := int(y*tmx.Width + x)
	if i < 0 || i >= len(layer.Content) {
		return zero, false
	}

	if layer.Content[i] == 0 {
		return zero, false
	}

	if tile, found := getTile(tmx, x, y, layer.Content[i]); found {
		layer.Tiles[idx] = tile
		return tile, true
	}

	return zero, false
}

func getChunkTileAt(tmx *Tmx, layer *TilemapLayer, x, y int32, layerIdx int) (TileData, bool) {
	var zero TileData

	idx := NewTileKey(x, y)
	if tile, exists := layer.Tiles[idx]; exists {
		return tile, true
	}

	for i := range layer.Chunks {
		chunk := &tmx.Layers[layerIdx].Data.Chunks[i]
		if x < chunk.X || x >= chunk.X+chunk.Width || y < chunk.Y || y >= chunk.Y+chunk.Height {
			continue
		}

		localX := x - chunk.X
		localY := y - chunk.Y
		localIdx := int(localY*chunk.Width + localX)
		if localIdx < 0 || localIdx >= len(layer.Chunks[i]) {
			return zero, false
		}

		if layer.Chunks[i][localIdx] == 0 {
			return zero, false
		}

		if tile, found := getTile(tmx, x, y, layer.Chunks[i][localIdx]); found {
			layer.Tiles[idx] = tile
			return tile, true
		}
	}

	return TileData{}, false
}

func getTile(tmx *Tmx, x, y int32, content uint32) (TileData, bool) {
	var zero TileData

	tileID, flags := DecodeGID(content)
	if tileID == 0 {
		return zero, false
	}

	_, tileID, tsIdx := TilesetByGID(tmx, tileID)
	if tsIdx == -1 {
		return zero, false
	}

	return TileData{
		TsIdx:    tsIdx,
		X:        x * tmx.TileWidth,
		Y:        y * tmx.TileHeight,
		TileID:   tileID,
		FlipFlag: flags,
	}, true
}

func calculateTileBounds(tmx *Tmx) (minX, minY, maxX, maxY int32) {
	if tmx.IsInfinite() {
		return calculateTileInfiniteBounds(tmx)
	}
	return 0, 0, tmx.Width * tmx.TileWidth, tmx.Height * tmx.TileHeight
}

func calculateTileInfiniteBounds(tmx *Tmx) (minX, minY, maxX, maxY int32) {
	minX = math.MaxInt32
	minY = math.MaxInt32
	maxX = math.MinInt32
	maxY = math.MinInt32

	for i := range tmx.Layers {
		for j := range tmx.Layers[i].Data.Chunks {
			minX = minInt32(minX, tmx.Layers[i].Data.Chunks[j].X)
			minY = minInt32(minY, tmx.Layers[i].Data.Chunks[j].Y)
			maxX = maxInt32(maxX, tmx.Layers[i].Data.Chunks[j].X+tmx.Layers[i].Data.Chunks[j].Width)
			maxY = maxInt32(maxY, tmx.Layers[i].Data.Chunks[j].Y+tmx.Layers[i].Data.Chunks[j].Height)
		}
	}

	minX *= tmx.TileWidth
	minY *= tmx.TileHeight
	maxX *= tmx.TileWidth
	maxY *= tmx.TileHeight
	return
}

func calculateQueryRegion(minX, minY, maxX, maxY, tileWidth, tileHeight int32) TileRegion {
	minX /= tileWidth
	minY /= tileHeight
	maxX = (maxX + tileWidth - 1) / tileWidth
	maxY = (maxY + tileHeight - 1) / tileHeight
	return TileRegion{
		MinX: minX - 1,
		MinY: minY - 1,
		MaxX: maxX,
		MaxY: maxY,
	}
}
