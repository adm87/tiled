package tilemap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/adm87/tiled"
	"github.com/adm87/utilities/hash"
)

// Test helpers
func createTestTmx(width, height, tileWidth, tileHeight int32, infinite bool) *tiled.Tmx {
	tmx := &tiled.Tmx{
		TileWidth:  tileWidth,
		TileHeight: tileHeight,
		Width:      width,
		Height:     height,
	}

	if infinite {
		tmx.Flags |= tiled.MapFlagInfinite
	}

	// Add a test tileset
	tmx.Tilesets = []tiled.Tileset{
		{
			Source:   "test.tsx",
			FirstGID: 1,
		},
	}

	if infinite {
		// Create infinite map with chunks
		tmx.Layers = []tiled.Layer{
			{
				Width:  width,
				Height: height,
				Flags:  tiled.LayerFlagVisible,
				Data: tiled.Data{
					Encoding:    tiled.EncodingCSV,
					Compression: tiled.CompressionNone,
					Chunks: []tiled.Chunk{
						{
							X:       0,
							Y:       0,
							Width:   16,
							Height:  16,
							Content: generateChunkData(16, 16),
						},
						{
							X:       16,
							Y:       0,
							Width:   16,
							Height:  16,
							Content: generateChunkData(16, 16),
						},
					},
				},
			},
		}
	} else {
		// Create single chunk map
		tmx.Layers = []tiled.Layer{
			{
				Width:  width,
				Height: height,
				Flags:  tiled.LayerFlagVisible,
				Data: tiled.Data{
					Encoding:    tiled.EncodingCSV,
					Compression: tiled.CompressionNone,
					Content:     generateChunkData(width, height),
				},
			},
		}
	}

	return tmx
}

func generateChunkData(width, height int32) string {
	// Generate simple test data (tile IDs 1-10) in CSV format
	var data []string
	for y := int32(0); y < height; y++ {
		var row []string
		for x := int32(0); x < width; x++ {
			tileID := (x+y)%10 + 1
			row = append(row, fmt.Sprintf("%d", tileID))
		}
		data = append(data, strings.Join(row, ","))
	}
	return strings.Join(data, ",")
}

// Unit Tests

func TestNewMap(t *testing.T) {
	m := NewMap()
	if m == nil {
		t.Fatal("NewMap() returned nil")
	}
	if m.Tmx != nil {
		t.Error("New map should have nil Tmx")
	}
	if len(m.layers) != 0 {
		t.Error("New map should have empty layers")
	}
}

func TestSetTmx(t *testing.T) {
	tests := []struct {
		name    string
		tmx     *tiled.Tmx
		wantErr bool
	}{
		{
			name:    "nil tmx",
			tmx:     nil,
			wantErr: true,
		},
		{
			name:    "empty layers",
			tmx:     &tiled.Tmx{Layers: []tiled.Layer{}},
			wantErr: true,
		},
		{
			name:    "valid single chunk",
			tmx:     createTestTmx(32, 32, 16, 16, false),
			wantErr: false,
		},
		{
			name:    "valid infinite map",
			tmx:     createTestTmx(32, 32, 16, 16, true),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMap()
			err := m.SetTmx(tt.tmx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetTmx() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && m.Tmx != tt.tmx {
				t.Error("SetTmx() did not set Tmx correctly")
			}
		})
	}
}

func TestFrame(t *testing.T) {
	m := NewMap()
	frame := m.Frame()

	if frame == nil {
		t.Fatal("Frame() returned nil")
	}

	// Test initial bounds
	minX, minY := frame.Min()
	maxX, maxY := frame.Max()
	if minX != 0 || minY != 0 || maxX != 0 || maxY != 0 {
		t.Error("Initial frame bounds should be (0,0,0,0)")
	}

	// Test setting bounds
	frame.Set(10, 20, 100, 200)
	minX, minY = frame.Min()
	maxX, maxY = frame.Max()
	if minX != 10 || minY != 20 || maxX != 100 || maxY != 200 {
		t.Errorf("Frame.Set() failed: got (%v,%v,%v,%v), want (10,20,100,200)", minX, minY, maxX, maxY)
	}

	// Test width/height
	if frame.Width() != 90 || frame.Height() != 180 {
		t.Errorf("Frame dimensions: got %vx%v, want 90x180", frame.Width(), frame.Height())
	}
}

func TestRegionEquals(t *testing.T) {
	r1 := Region{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10}
	r2 := Region{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10}
	r3 := Region{MinX: 1, MinY: 0, MaxX: 10, MaxY: 10}

	if !r1.Equals(&r2) {
		t.Error("Equal regions should return true")
	}
	if r1.Equals(&r3) {
		t.Error("Different regions should return false")
	}
}

func TestBufferFrame(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Map
		wantErr bool
	}{
		{
			name: "no tmx data",
			setup: func() *Map {
				return NewMap()
			},
			wantErr: true,
		},
		{
			name: "valid single chunk",
			setup: func() *Map {
				m := NewMap()
				tmx := createTestTmx(32, 32, 16, 16, false)
				m.SetTmx(tmx)
				m.Frame().Set(0, 0, 256, 256) // 16x16 tiles at 16px each
				return m
			},
			wantErr: false,
		},
		{
			name: "valid infinite map",
			setup: func() *Map {
				m := NewMap()
				tmx := createTestTmx(32, 32, 16, 16, true)
				m.SetTmx(tmx)
				m.Frame().Set(0, 0, 512, 256) // Spans both chunks
				return m
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			err := m.BufferFrame()
			if (err != nil) != tt.wantErr {
				t.Errorf("BufferFrame() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIterator(t *testing.T) {
	m := NewMap()
	tmx := createTestTmx(16, 16, 16, 16, false)
	err := m.SetTmx(tmx)
	if err != nil {
		t.Fatal("Failed to set tmx:", err)
	}

	m.Frame().Set(0, 0, 256, 256) // Full map
	err = m.BufferFrame()
	if err != nil {
		t.Fatal("Failed to buffer frame:", err)
	}

	iter := m.Itr()
	layerCount := 0
	totalTiles := 0

	for {
		tiles := iter.Next()
		if tiles == nil {
			break
		}
		layerCount++
		totalTiles += len(tiles)
	}

	if layerCount != 1 {
		t.Errorf("Expected 1 layer, got %d", layerCount)
	}
	if totalTiles == 0 {
		t.Error("Expected some tiles, got 0")
	}
}

func TestChunkPooling(t *testing.T) {
	// Test that chunks are properly pooled
	m := NewMap()
	tmx := createTestTmx(16, 16, 16, 16, false)

	// Set and flush multiple times to test pooling
	for i := 0; i < 5; i++ {
		err := m.SetTmx(tmx)
		if err != nil {
			t.Fatal("Failed to set tmx:", err)
		}
		m.Flush()
	}

	// If we get here without panics, pooling is working
}

func TestLayerPooling(t *testing.T) {
	// Test that layers are properly pooled
	m := NewMap()
	tmx := createTestTmx(16, 16, 16, 16, true) // Infinite map with multiple chunks

	// Set and flush multiple times to test pooling
	for i := 0; i < 5; i++ {
		err := m.SetTmx(tmx)
		if err != nil {
			t.Fatal("Failed to set tmx:", err)
		}
		m.Flush()
	}

	// If we get here without panics, pooling is working
}

func TestConcurrentAccess(t *testing.T) {
	m := NewMap()
	tmx := createTestTmx(32, 32, 16, 16, false)
	err := m.SetTmx(tmx)
	if err != nil {
		t.Fatal("Failed to set tmx:", err)
	}

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			m.Frame().Set(0, 0, 256, 256)
			m.BufferFrame()
			iter := m.Itr()
			for iter.Next() != nil {
				// Just iterate
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTmxSwapping(t *testing.T) {
	m := NewMap()

	// Create first TMX - single chunk map
	tmx1 := createTestTmx(32, 32, 16, 16, false)
	err := m.SetTmx(tmx1)
	if err != nil {
		t.Fatal("Failed to set first tmx:", err)
	}

	// Buffer several frames to build cache
	frames := []struct{ x, y, w, h float32 }{
		{0, 0, 256, 256},     // Top-left quadrant
		{128, 128, 384, 384}, // Center area
		{256, 0, 512, 256},   // Top-right quadrant
		{0, 256, 256, 512},   // Bottom-left quadrant
	}

	for i, frame := range frames {
		m.Frame().Set(frame.x, frame.y, frame.x+frame.w, frame.y+frame.h)
		err = m.BufferFrame()
		if err != nil {
			t.Fatalf("Failed to buffer frame %d: %v", i, err)
		}

		// Verify we have cached data
		iter := m.Itr()
		layerCount := 0
		totalTiles := 0
		for {
			tiles := iter.Next()
			if tiles == nil {
				break
			}
			layerCount++
			totalTiles += len(tiles)
		}
		if totalTiles == 0 {
			t.Errorf("Frame %d: Expected some tiles, got 0", i)
		}
	}

	// Verify cache is populated
	if len(m.cachedData) == 0 {
		t.Error("Expected cached data after buffering frames")
	}
	if len(m.cachedPositions) == 0 {
		t.Error("Expected cached positions after buffering frames")
	}

	// Create second TMX - infinite map with different structure
	tmx2 := createTestTmx(64, 64, 32, 32, true) // Different tile size and infinite
	// Add more chunks to make it different
	for x := int32(0); x < 4; x++ {
		for y := int32(0); y < 4; y++ {
			if x == 0 && y == 0 || x == 1 && y == 0 {
				continue // Skip existing chunks
			}
			tmx2.Layers[0].Data.Chunks = append(tmx2.Layers[0].Data.Chunks, tiled.Chunk{
				X:       x * 16,
				Y:       y * 16,
				Width:   16,
				Height:  16,
				Content: generateChunkData(16, 16),
			})
		}
	}

	// Record state before swap
	oldLayerCount := len(m.layers)
	oldCacheSize := len(m.cachedData)

	// Swap to second TMX
	err = m.SetTmx(tmx2)
	if err != nil {
		t.Fatal("Failed to set second tmx:", err)
	}

	// Verify old cache was cleared
	if len(m.cachedData) != 0 {
		t.Error("Expected cached data to be cleared after TMX swap")
	}
	if len(m.cachedPositions) != 0 {
		t.Error("Expected cached positions to be cleared after TMX swap")
	}

	// Verify new layer structure
	newLayerCount := len(m.layers)
	if newLayerCount == 0 {
		t.Error("Expected layers after TMX swap")
	}

	// Verify TMX reference was updated
	if m.Tmx != tmx2 {
		t.Error("Expected TMX reference to be updated")
	}

	// Test that the new map works correctly
	m.Frame().Set(0, 0, 512, 512) // Larger frame for infinite map
	err = m.BufferFrame()
	if err != nil {
		t.Fatal("Failed to buffer frame after TMX swap:", err)
	}

	// Verify new cache is built
	if len(m.cachedData) == 0 {
		t.Error("Expected new cached data after buffering new TMX")
	}

	// Test iteration works with new TMX
	iter := m.Itr()
	layerCount := 0
	totalTiles := 0
	for {
		tiles := iter.Next()
		if tiles == nil {
			break
		}
		layerCount++
		totalTiles += len(tiles)
	}
	if layerCount == 0 {
		t.Error("Expected layers in new TMX")
	}

	// Test another swap back to single chunk
	tmx3 := createTestTmx(16, 16, 8, 8, false) // Even smaller
	err = m.SetTmx(tmx3)
	if err != nil {
		t.Fatal("Failed to set third tmx:", err)
	}

	// Verify everything still works - use larger frame to ensure we capture tiles
	m.Frame().Set(0, 0, 256, 256) // Use larger frame than map size
	err = m.BufferFrame()
	if err != nil {
		t.Fatal("Failed to buffer frame after second TMX swap:", err)
	}

	iter = m.Itr()
	finalLayerCount := 0
	finalTotalTiles := 0
	for {
		tiles := iter.Next()
		if tiles == nil {
			break
		}
		finalLayerCount++
		finalTotalTiles += len(tiles)
	}

	// Debug output
	t.Logf("Final TMX: %dx%d tiles, %dx%d pixels per tile", tmx3.Width, tmx3.Height, tmx3.TileWidth, tmx3.TileHeight)
	t.Logf("Map dimensions: %dx%d pixels", tmx3.Width*tmx3.TileWidth, tmx3.Height*tmx3.TileHeight)
	t.Logf("Query frame: 0,0 to 256,256")
	t.Logf("Layers found: %d, Total tiles: %d", finalLayerCount, finalTotalTiles)

	if finalLayerCount == 0 {
		t.Error("Expected layers in final TMX")
	}

	t.Logf("TMX swap test completed successfully:")
	t.Logf("  - Initial layers: %d, cache size: %d", oldLayerCount, oldCacheSize)
	t.Logf("  - Second TMX layers: %d", newLayerCount)
	t.Logf("  - Final TMX layers: %d, tiles: %d", finalLayerCount, finalTotalTiles)
}

// Benchmarks

func BenchmarkSetTmx(b *testing.B) {
	tmx := createTestTmx(64, 64, 16, 16, false)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m := NewMap()
		m.SetTmx(tmx)
	}
}

func BenchmarkBufferFrame(b *testing.B) {
	m := NewMap()
	tmx := createTestTmx(64, 64, 16, 16, false)
	m.SetTmx(tmx)
	m.Frame().Set(0, 0, 512, 512) // 32x32 tiles

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.BufferFrame()
	}
}

func BenchmarkBufferFrameInfinite(b *testing.B) {
	m := NewMap()
	tmx := createTestTmx(32, 32, 16, 16, true)
	m.SetTmx(tmx)
	m.Frame().Set(0, 0, 512, 256) // Spans multiple chunks

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.BufferFrame()
	}
}

func BenchmarkIterator(b *testing.B) {
	m := NewMap()
	tmx := createTestTmx(64, 64, 16, 16, false)
	m.SetTmx(tmx)
	m.Frame().Set(0, 0, 1024, 1024) // Full map
	m.BufferFrame()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		iter := m.Itr()
		for iter.Next() != nil {
			// Just iterate
		}
	}
}

func BenchmarkSpatialQuery(b *testing.B) {
	m := NewMap()
	tmx := createTestTmx(128, 128, 16, 16, true) // Large infinite map

	// Add many chunks
	for x := int32(0); x < 8; x++ {
		for y := int32(0); y < 8; y++ {
			tmx.Layers[0].Data.Chunks = append(tmx.Layers[0].Data.Chunks, tiled.Chunk{
				X:       x * 16,
				Y:       y * 16,
				Width:   16,
				Height:  16,
				Content: generateChunkData(16, 16),
			})
		}
	}

	m.SetTmx(tmx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Query small viewport (simulating camera movement)
		frameX := float32(i%1000) * 2 // Moving viewport
		frameY := float32(i%1000) * 2
		m.Frame().Set(frameX, frameY, frameX+320, frameY+240)
		m.BufferFrame()
	}
}

func BenchmarkChunkPooling(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Get chunk from pool
		chunk := chunkPool.Get().(*Chunk)

		// Simulate chunk usage
		chunk.x, chunk.y = int32(i%100), int32(i%100)
		chunk.w, chunk.h = 16, 16
		chunk.raw = "test data"

		// Simulate some tile caching
		if chunk.tiles != nil {
			key := hash.EncodeGridKey(1, 1)
			chunk.tiles[key] = Data{X: 16, Y: 16, TileID: 1, TsIdx: 0}
		}

		// Return to pool
		chunk.Flush()
		chunkPool.Put(chunk)
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	m := NewMap()
	tmx := createTestTmx(256, 256, 16, 16, false) // Large map
	m.SetTmx(tmx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate camera movement across the map
		x := float32(i % 1000)
		y := float32(i % 1000)
		m.Frame().Set(x, y, x+640, y+480)
		m.BufferFrame()

		// Force iteration to measure full memory impact
		iter := m.Itr()
		for iter.Next() != nil {
			// Process tiles
		}
	}
}

func BenchmarkTmxSwapping(b *testing.B) {
	m := NewMap()

	// Create different TMX maps to swap between
	tmx1 := createTestTmx(32, 32, 16, 16, false)
	tmx2 := createTestTmx(64, 64, 32, 32, true)
	tmx3 := createTestTmx(16, 16, 8, 8, false)

	tmxMaps := []*tiled.Tmx{tmx1, tmx2, tmx3}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Swap between different TMX maps
		tmx := tmxMaps[i%len(tmxMaps)]
		m.SetTmx(tmx)

		// Buffer a frame to test full cycle
		m.Frame().Set(0, 0, 256, 256)
		m.BufferFrame()
	}
}

func BenchmarkCacheInvalidation(b *testing.B) {
	m := NewMap()
	tmx1 := createTestTmx(64, 64, 16, 16, false)
	tmx2 := createTestTmx(64, 64, 16, 16, true)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Set first TMX and build cache
		m.SetTmx(tmx1)
		m.Frame().Set(0, 0, 512, 512)
		m.BufferFrame()

		// Swap TMX (should invalidate cache)
		m.SetTmx(tmx2)
		m.Frame().Set(100, 100, 612, 612)
		m.BufferFrame()
	}
}
