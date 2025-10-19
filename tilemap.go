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

func (tr TileRegion) Overlaps(other TileRegion) bool {
	return tr.MinX < other.MaxX && tr.MaxX > other.MinX &&
		tr.MinY < other.MaxY && tr.MaxY > other.MinY
}

func (tr TileRegion) Width() int32 {
	return tr.MaxX - tr.MinX
}

func (tr TileRegion) Height() int32 {
	return tr.MaxY - tr.MinY
}

// CompatibleForCaching returns true if regions have compatible dimensions for cache reuse.
func (tr TileRegion) CompatibleForCaching(other TileRegion) bool {
	widthDiff := tr.Width() - other.Width()
	heightDiff := tr.Height() - other.Height()
	return widthDiff >= -1 && widthDiff <= 1 && heightDiff >= -1 && heightDiff <= 1
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

	cachedTileRegion TileRegion // Cached tile region for current query
	cachedTileData   []TileData // Cached tile data for current query
	cachedPositions  []int      // Cached positions for current query

	minX, minY int32 // Minimum tile coordinates boundary
	maxX, maxY int32 // Maximum tile coordinates boundary

	decodedLayers []TilemapLayer
}

func NewTilemap() *Tilemap {
	return &Tilemap{
		Tmx:              nil,
		cachedTileRegion: TileRegion{},
		cachedTileData:   make([]TileData, 0, 64),    // Pre-allocate some capacity
		cachedPositions:  make([]int, 0, 8),          // Pre-allocate for typical layer count
		decodedLayers:    make([]TilemapLayer, 0, 4), // Pre-allocate for typical layer count
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
	tm.cachedTileRegion = TileRegion{}
	tm.cachedTileData = tm.cachedTileData[:0]
	tm.cachedPositions = tm.cachedPositions[:0]
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
// Returns an error if the tilemap has no Tmx data set or if coordinates are invalid.
func (tm *Tilemap) GetTiles(minX, minY, maxX, maxY float32) (TileIterator, error) {
	if tm.Tmx == nil || len(tm.decodedLayers) == 0 {
		return TileIterator{}, ErrNoTmxData
	}

	if minX > maxX || minY > maxY {
		return TileIterator{}, errors.New("invalid coordinate bounds: min > max")
	}

	queryRegion := calculateQueryRegion(minX, minY, maxX, maxY, tm.Tmx.TileWidth, tm.Tmx.TileHeight)
	if queryRegion.Equal(tm.cachedTileRegion) {
		return tm.buildIterator(), nil
	}

	if queryRegion.CompatibleForCaching(tm.cachedTileRegion) {
		tm.updateCache(queryRegion)
		return tm.buildIterator(), nil
	}

	size := int(queryRegion.Width() * queryRegion.Height() * int32(len(tm.decodedLayers)))
	if cap(tm.cachedTileData) < size {
		tm.cachedTileData = make([]TileData, 0, size)
	}

	tm.updateCache(queryRegion)
	return tm.buildIterator(), nil
}

func (tm *Tilemap) updateCache(region TileRegion) {
	tm.cachedTileRegion = region

	tm.cachedTileData = tm.cachedTileData[:0]
	tm.cachedPositions = tm.cachedPositions[:0]

	for i := range tm.decodedLayers {
		tm.cachedPositions = append(tm.cachedPositions, len(tm.cachedTileData))

		if !tm.Tmx.Layers[i].IsVisible() {
			continue
		}

		for y := region.MinY; y < region.MaxY; y++ {
			for x := region.MinX; x < region.MaxX; x++ {
				if tile, found := getTileAt(tm.Tmx, &tm.decodedLayers[i], x, y, i); found {
					tm.cachedTileData = append(tm.cachedTileData, tile)
				}
			}
		}
	}

	tm.cachedPositions = append(tm.cachedPositions, len(tm.cachedTileData))
}

func (tm *Tilemap) buildIterator() TileIterator {
	iteratorTiles := make([]TileData, len(tm.cachedTileData))
	copy(iteratorTiles, tm.cachedTileData)

	iteratorPositions := make([]int, len(tm.cachedPositions))
	copy(iteratorPositions, tm.cachedPositions)

	return TileIterator{iteratorTiles, iteratorPositions, 0}
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

	i := int64(y)*int64(tmx.Width) + int64(x)
	if i < 0 || i >= int64(len(layer.Content)) {
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
		localIdx := int64(localY)*int64(chunk.Width) + int64(localX)
		if localIdx < 0 || localIdx >= int64(len(layer.Chunks[i])) {
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

func calculateQueryRegion(minX, minY, maxX, maxY float32, tileWidth, tileHeight int32) TileRegion {
	return TileRegion{
		MinX: int32(math.Floor(float64(minX) / float64(tileWidth))),
		MinY: int32(math.Floor(float64(minY) / float64(tileHeight))),
		MaxX: int32(math.Ceil(float64(maxX) / float64(tileWidth))),
		MaxY: int32(math.Ceil(float64(maxY) / float64(tileHeight))),
	}
}
