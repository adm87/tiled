package main

import (
	_ "net/http/pprof"
	"time"

	"log"
	"math/rand"
	"net/http"

	"github.com/adm87/tiled"
	"github.com/adm87/tiled/examples/shared"
)

var (
	loadedTmx = make([]*tiled.Tmx, 0)
	testTmx   = 0
)

type Rect struct {
	X, Y          int32
	Width, Height int32
}

func (r Rect) Bounds() (minX, minY, maxX, maxY int32) {
	return r.X, r.Y, r.X + r.Width, r.Y + r.Height
}

func main() {
	loadedTmx = append(loadedTmx, shared.MustLoadTiledAsset[tiled.Tmx](shared.TilemapExampleA))
	loadedTmx = append(loadedTmx, shared.MustLoadTiledAsset[tiled.Tmx](shared.TilemapExampleB))

	tilemap := tiled.NewTilemap()
	tilemap.SetTmx(loadedTmx[testTmx])

	region := Rect{0, 0, 200, 200}

	go func() {
		log.Println("Profiling server at http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	now := time.Now()

	// Profiling loop - run as fast as possible to stress test
	for {
		minX, minY, maxX, maxY := tilemap.Bounds()

		// Random region movement to prevent cache optimization
		region.X = minX + int32(rand.Int31n(maxX-minX-region.Width))
		region.Y = minY + int32(rand.Int31n(maxY-minY-region.Height))

		// Exercise GetTiles - the main performance target
		itr, err := tilemap.GetTiles(region.Bounds())
		if err != nil {
			log.Fatal(err)
		}

		// Iterate through all tiles to stress the iterator
		for tiles := itr.Next(); tiles != nil; tiles = itr.Next() {
			for _, tile := range tiles {
				// Exercise tileset lookups
				_, err := tilemap.GetTileset(tile.TsIdx)
				if err != nil {
					continue
				}

				// Exercise flip flag operations
				_ = tile.FlipFlag.Horizontal()
				_ = tile.FlipFlag.Vertical()
			}
		}

		if time.Since(now) > 5*time.Second {
			testTmx = (testTmx + 1) % len(loadedTmx)
			tilemap.SetTmx(loadedTmx[testTmx])
			now = time.Now()
			log.Println("Switched tilemap for profiling")
		}
	}
}
