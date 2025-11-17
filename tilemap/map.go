package tilemap

import (
	"errors"
	"math"
	"sync"

	"github.com/adm87/tiled"
	"github.com/adm87/utilities/hash"
)

var (
	ErrNoTmxData       = errors.New("no Tmx data set")
	ErrInvalidTmxData  = errors.New("invalid Tmx data")
	ErrTilesetNotFound = errors.New("tileset not found")
	ErrTileNotFound    = errors.New("tile not found")
	ErrTilesetSource   = errors.New("tileset source is empty")
)

const (
	DefaultChunkSize int32 = 16 // in tiles
)

// ====================== Region =====================

// Region represents a rectangular region in tile coordinates.
type Region struct {
	MinX, MinY int32
	MaxX, MaxY int32
}

func (r *Region) Equals(other *Region) bool {
	return r.MinX == other.MinX &&
		r.MinY == other.MinY &&
		r.MaxX == other.MaxX &&
		r.MaxY == other.MaxY
}

// ====================== Data =====================

type Data struct {
	X, Y     float32        // World position
	TileID   uint32         // Tile ID
	TsIdx    int            // Tileset index
	FlipFlag tiled.FlipFlag // Flip flags
}

// ====================== Chunk =====================

var chunkPool = sync.Pool{
	New: func() any {
		return &Chunk{
			data:  make([]uint32, 0),
			tiles: make(map[uint64]Data),
		}
	},
}

type Chunk struct {
	x, y        int32
	w, h        int32
	isDecoded   bool
	encoding    tiled.Encoding
	compression tiled.Compression
	raw         string
	data        []uint32
	tiles       map[uint64]Data
}

func (c *Chunk) Flush() {
	clear(c.tiles)
}

// ====================== Layer =====================

var layerPool = sync.Pool{
	New: func() any {
		return &Layer{
			Grid: hash.NewGrid[*Chunk](0, 0),
		}
	},
}

type Layer struct {
	*hash.Grid[*Chunk]
}

func (l *Layer) Flush() {
	if l.Grid != nil {
		l.Grid.ForEach(func(chunk *Chunk) {
			chunk.Flush()
			chunkPool.Put(chunk)
		})
		l.Grid.Clear()
	}
}

// ====================== Iterator =====================

// Iterator provides a way to iterate over tiles in the visible frame of a tilemap.
type Iterator struct {
	tiles  []Data
	layers []int
	index  int
}

func (it *Iterator) Next() []Data {
	if it.index >= len(it.layers)-1 {
		return nil
	}

	start := it.layers[it.index]
	end := it.layers[it.index+1]
	it.index++

	return it.tiles[start:end]
}

// ====================== Frame =====================

// Frame represents the visible region of a tilemap in world coordinates.
type Frame struct {
	bounds [4]float32
}

func (f *Frame) Width() float32 {
	return f.bounds[2] - f.bounds[0]
}

func (f *Frame) Height() float32 {
	return f.bounds[3] - f.bounds[1]
}

func (f *Frame) Min() (x, y float32) {
	return f.bounds[0], f.bounds[1]
}

func (f *Frame) Max() (x, y float32) {
	return f.bounds[2], f.bounds[3]
}

func (f *Frame) Bounds() (minX, minY, maxX, maxY float32) {
	return f.bounds[0], f.bounds[1], f.bounds[2], f.bounds[3]
}

func (f *Frame) Set(frame [4]float32) {
	f.bounds = frame
}

// ====================== Map =====================

func init() {
	// Seed the chunk pool with a couple of chunks
	chunkPool.Put(chunkPool.Get())
	chunkPool.Put(chunkPool.Get())

	// Seed the layer pool with a couple of layers
	layerPool.Put(layerPool.Get())
	layerPool.Put(layerPool.Get())
}

// Map represents the decoded tilemap data from a Tmx file.
//
// It provides methods to retrieve tile data, manage layers, and buffer the map for rendering.
type Map struct {
	Tmx    *tiled.Tmx
	layers []*Layer

	frame Frame // current frame

	cachedRegion    Region
	cachedData      []Data
	cachedPositions []int
}

func NewMap() *Map {
	return &Map{
		Tmx: nil,
		frame: Frame{
			bounds: [4]float32{0, 0, 0, 0},
		},
		layers: make([]*Layer, 0, 4),
	}
}

// Itr returns an iterator for the map.
// Use this for iterating over tiles in the visible frame.
func (tm *Map) Itr() Iterator {
	return Iterator{
		tiles:  tm.cachedData,
		layers: tm.cachedPositions,
		index:  0,
	}
}

// Frame returns the visible region of the tilemap in world coordinates.
// Use this to get or set the visible region of the map.
//
// Frame only returns the dimensions of the visible region of the tilemap.
// It does not update or buffer the map for rendering.
func (tm *Map) Frame() *Frame {
	return &tm.frame
}

// Flush clears all layers and their chunks from the map.
func (tm *Map) Flush() {
	tm.flush()
}

// BufferFrame buffers tile data for current frame.
func (tm *Map) BufferFrame() error {
	if tm.Tmx == nil {
		return ErrNoTmxData
	}

	if len(tm.layers) == 0 {
		return ErrInvalidTmxData
	}

	region := tm.computeTileRegion()
	if region.Equals(&tm.cachedRegion) {
		return nil
	}

	width := region.MaxX - region.MinX
	height := region.MaxY - region.MinY

	size := int(width*height) * len(tm.layers)
	if cap(tm.cachedData) < size {
		tm.cachedData = make([]Data, 0, size)
	}

	return tm.updateCache(region)
}

// SetTmx sets the Tmx data for the map and builds the underlying structures of the map.
// Setting a new Tmx will clear any existing layers data, but will not reset the frame.
func (tm *Map) SetTmx(tmx *tiled.Tmx) error {
	if tmx == nil || len(tmx.Layers) == 0 {
		return ErrInvalidTmxData
	}

	tm.flush()
	tm.Tmx = tmx

	return tm.buildLayers()
}

func (tm *Map) GetTileset(index int) (*tiled.Tileset, error) {
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

func (tm *Map) flush() {
	for i := range tm.layers {
		if tm.layers[i] != nil {
			tm.layers[i].Flush()
			layerPool.Put(tm.layers[i])
		}
	}
	tm.layers = tm.layers[:0]
	tm.cachedData = tm.cachedData[:0]
	tm.cachedPositions = tm.cachedPositions[:0]
}

func (tm *Map) buildLayers() error {
	for i := range tm.Tmx.Layers {
		if tm.Tmx.IsInfinite() {
			tm.multiChunklayer(&tm.Tmx.Layers[i], tm.Tmx.TileWidth, tm.Tmx.TileHeight)
		} else {
			tm.singleChunkLayer(&tm.Tmx.Layers[i], tm.Tmx.TileWidth, tm.Tmx.TileHeight)
		}
	}
	return nil
}

func (tm *Map) multiChunklayer(data *tiled.Layer, tileWidth, tileHeight int32) {
	width := data.Data.Chunks[0].Width * tileWidth
	height := data.Data.Chunks[0].Height * tileHeight

	layer := layerPool.Get().(*Layer)
	layer.Grid.Resize(float32(width), float32(height))

	for _, c := range data.Data.Chunks {
		chunk := chunkPool.Get().(*Chunk)
		chunk.raw = c.Content

		minX := float32(c.X * tileWidth)
		minY := float32(c.Y * tileHeight)
		maxX := float32((c.X + c.Width) * tileWidth)
		maxY := float32((c.Y + c.Height) * tileHeight)

		chunk.x, chunk.y = c.X, c.Y
		chunk.w, chunk.h = c.Width, c.Height

		layer.Grid.Insert(chunk, [4]float32{minX, minY, maxX, maxY}, hash.NoGridPadding)
	}

	tm.layers = append(tm.layers, layer)
}

func (tm *Map) singleChunkLayer(data *tiled.Layer, tileWidth, tileHeight int32) {
	width := data.Width * tileWidth
	height := data.Height * tileHeight

	layer := layerPool.Get().(*Layer)
	layer.Grid.Resize(float32(width), float32(height))

	chunk := chunkPool.Get().(*Chunk)
	chunk.raw = data.Data.Content
	chunk.x, chunk.y = 0, 0
	chunk.w, chunk.h = data.Width, data.Height

	layer.Grid.Insert(chunk, [4]float32{0, 0, float32(width), float32(height)}, hash.NoGridPadding)
	tm.layers = append(tm.layers, layer)
}

func (tm *Map) updateCache(region Region) error {
	tm.cachedRegion = region

	tm.cachedData = tm.cachedData[:0]
	tm.cachedPositions = tm.cachedPositions[:0]

	for i := range tm.layers {
		tm.cachedPositions = append(tm.cachedPositions, len(tm.cachedData))

		if tm.Tmx.Layers[i].IsVisible() {
			chunks := tm.layers[i].Grid.Query([4]float32{
				float32(region.MinX) * float32(tm.Tmx.TileWidth),
				float32(region.MinY) * float32(tm.Tmx.TileHeight),
				float32(region.MaxX) * float32(tm.Tmx.TileWidth),
				float32(region.MaxY) * float32(tm.Tmx.TileHeight),
			})
			for j := range chunks {
				sX := max(region.MinX, chunks[j].x)
				sY := max(region.MinY, chunks[j].y)
				eX := min(region.MaxX, chunks[j].x+chunks[j].w)
				eY := min(region.MaxY, chunks[j].y+chunks[j].h)

				for x := sX; x < eX; x++ {
					for y := sY; y < eY; y++ {
						if tile, ok := tm.getTileFromChunk(chunks[j], x, y); ok {
							tm.cachedData = append(tm.cachedData, tile)
						}
					}
				}
			}
		}
	}

	tm.cachedPositions = append(tm.cachedPositions, len(tm.cachedData))
	return nil
}

func (tm *Map) getTileFromChunk(chunk *Chunk, x, y int32) (Data, bool) {
	var zero Data

	if x < chunk.x || x >= chunk.x+chunk.w || y < chunk.y || y >= chunk.y+chunk.h {
		return zero, false
	}

	if !chunk.isDecoded {
		data, err := tiled.DecodeContent(chunk.raw, chunk.encoding, chunk.compression)
		if err != nil {
			return Data{}, false
		}
		chunk.data = data
		chunk.isDecoded = true
	}

	localx := x - chunk.x
	localy := y - chunk.y

	key := hash.EncodeGridKey(localx, localy)
	if tile, ok := chunk.tiles[key]; ok {
		return tile, true
	}

	i := localy*(chunk.w) + localx
	if i < 0 || i >= int32(len(chunk.data)) {
		return zero, false
	}

	x = localx * tm.Tmx.TileWidth
	y = localy * tm.Tmx.TileHeight

	return GetTileData(chunk.data[i], tm.Tmx, float32(x), float32(y))
}

func (tm *Map) computeTileRegion() Region {
	minX, minY, maxX, maxY := tm.frame.Bounds()
	return Region{
		MinX: int32(math.Floor(float64(minX) / float64(tm.Tmx.TileWidth))),
		MinY: int32(math.Floor(float64(minY) / float64(tm.Tmx.TileHeight))),
		MaxX: int32(math.Ceil(float64(maxX) / float64(tm.Tmx.TileWidth))),
		MaxY: int32(math.Ceil(float64(maxY) / float64(tm.Tmx.TileHeight))),
	}
}

func GetTileData(gid uint32, tmx *tiled.Tmx, x, y float32) (Data, bool) {
	var zero Data

	tileID, flipFlags := tiled.DecodeGID(gid)
	if tileID == 0 {
		return zero, false
	}

	_, tileID, tsIdx := tiled.TilesetByGID(tmx, tileID)
	if tsIdx == -1 {
		return zero, false
	}

	return Data{
		TsIdx:    tsIdx,
		TileID:   tileID,
		FlipFlag: flipFlags,
		X:        x,
		Y:        y,
	}, true
}
