package main

import (
	"bytes"
	"image"
	"math"

	"github.com/adm87/tiled"
	"github.com/adm87/tiled/examples/shared"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 800 * 0.3
	screenHeight = 600 * 0.3
)

type Camera struct {
	X, Y          int32
	Width, Height int32
	Zoom          float64
}

func (c *Camera) Viewport() (minX, minY, maxX, maxY int32) {
	halfW := float64(c.Width) / (2 * c.Zoom)
	halfH := float64(c.Height) / (2 * c.Zoom)
	left := float64(c.X) - halfW
	top := float64(c.Y) - halfH
	return int32(left), int32(top), int32(left + 2*halfW), int32(top + 2*halfH)
}

func (c *Camera) ViewMatrix() ebiten.GeoM {
	m := ebiten.GeoM{}
	m.Translate(-float64(c.X), -float64(c.Y))
	m.Scale(c.Zoom, c.Zoom)
	m.Translate(float64(c.Width)/2, float64(c.Height)/2)
	return m
}

func (c *Camera) ClampToMapBounds(mapMinX, mapMinY, mapMaxX, mapMaxY int32) {
	halfW := float64(c.Width) / (2 * c.Zoom)
	halfH := float64(c.Height) / (2 * c.Zoom)
	c.X = int32(math.Max(float64(mapMinX)+halfW, math.Min(float64(c.X), float64(mapMaxX)-halfW)))
	c.Y = int32(math.Max(float64(mapMinY)+halfH, math.Min(float64(c.Y), float64(mapMaxY)-halfH)))
}

type Game struct {
	tilemap    *tiled.Tilemap
	camera     Camera
	op         ebiten.DrawImageOptions
	currentMap int
}

var (
	loadedTmx = make([]*tiled.Tmx, 0)
	loadedTsx = make(map[string]*tiled.Tsx)
	loadedImg = make(map[string]*ebiten.Image)
)

func NewGame() *Game {
	return &Game{
		camera: Camera{
			X:      0,
			Y:      0,
			Width:  screenWidth,
			Height: screenHeight,
			Zoom:   1,
		},
		tilemap: tiled.NewTilemap(),
		op:      ebiten.DrawImageOptions{},
	}
}

func main() {
	loadedTmx = append(loadedTmx, shared.MustLoadTiledAsset[tiled.Tmx](shared.TilemapExampleA))
	loadedTmx = append(loadedTmx, shared.MustLoadTiledAsset[tiled.Tmx](shared.TilemapExampleB))

	loadedTsx[shared.TilesetCharacters] = shared.MustLoadTiledAsset[tiled.Tsx](shared.TilesetCharacters)
	loadedTsx[shared.TilesetTiles] = shared.MustLoadTiledAsset[tiled.Tsx](shared.TilesetTiles)

	loadedImg[shared.TilemapPacked] = mustLoadImage(shared.TilemapPacked)
	loadedImg[shared.TilemapCharactersPacked] = mustLoadImage(shared.TilemapCharactersPacked)

	game := NewGame()
	// A Tmx reference must be set in the tilemap before using it.
	// The tilemap will panic if it doesn't have a Tmx before attempting to draw tiles.
	game.tilemap.SetTmx(loadedTmx[0])

	minX, minY, maxX, maxY := game.tilemap.Bounds()
	game.camera.X = (minX + maxX) / 2
	game.camera.Y = (minY + maxY) / 2

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func mustLoadImage(filename string) *ebiten.Image {
	data := shared.MustLoadImageAsset(filename)
	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	return img
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camera.X -= 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camera.X += 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camera.Y -= 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camera.Y += 4
	}

	if _, y := ebiten.Wheel(); y != 0 {
		g.camera.Zoom += float64(y) * 0.1
		if g.camera.Zoom < 1 {
			g.camera.Zoom = 1
		}
		if g.camera.Zoom > 4 {
			g.camera.Zoom = 4
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.currentMap = (g.currentMap + 1) % len(loadedTmx)
		// Reuse an existing tilemap to avoid allocations.
		// The tilemap.SetTmx method will clear any existing data.
		// This is more efficient than creating a new tilemap each time.
		g.tilemap.SetTmx(loadedTmx[g.currentMap])
	}

	g.camera.ClampToMapBounds(g.tilemap.Bounds())

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// GetTiles() returns an iterator that yields tiles for each layer in the Tmx.
	// Calls to Next() will return the tiles for the next layer. If nil is returned,
	// there are no more layers.
	// This allows drawing tiles in layer order without needing to sort them manually.
	// Layer order is determined by the order they are defined within the Tmx file.
	itr, err := g.tilemap.GetTiles(g.camera.Viewport())
	if err != nil {
		panic(err)
	}
	for tiles := itr.Next(); tiles != nil; tiles = itr.Next() {
		for _, tile := range tiles {
			g.DrawTile(screen, &tile)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) DrawTile(screen *ebiten.Image, tile *tiled.TileData) {
	tileset, err := g.tilemap.GetTileset(tile.TsIdx)
	if err != nil {
		println(err.Error())
		return
	}

	tsx, exists := loadedTsx[tileset.Source]
	if !exists {
		println("missing tsx: " + tileset.Source)
		return
	}

	img, exists := loadedImg[tsx.Image.Source]
	if !exists {
		println("missing image: " + tsx.Image.Source)
		return
	}

	srcX := (int32(tile.TileID) % tsx.Columns) * tsx.TileWidth
	srcY := (int32(tile.TileID) / tsx.Columns) * tsx.TileHeight
	srcRect := image.Rect(int(srcX), int(srcY), int(srcX+tsx.TileWidth), int(srcY+tsx.TileHeight))

	distX := float64(tile.X) + float64(tsx.TileOffset.X)
	distY := float64(tile.Y) + float64(tsx.TileOffset.Y)
	distY -= float64(tsx.TileHeight) - float64(g.tilemap.Tmx.TileHeight) // Align to bottom of tile

	g.op.GeoM.Reset()

	if tile.FlipFlag&tiled.FlipDiagonal != 0 {
		g.op.GeoM.Rotate(math.Pi * 0.5)
		g.op.GeoM.Scale(-1, 1)
		g.op.GeoM.Translate(float64(tsx.TileHeight-tsx.TileWidth), 0)
	}

	if tile.FlipFlag&tiled.FlipHorizontal != 0 {
		g.op.GeoM.Scale(-1, 1)
		g.op.GeoM.Translate(float64(tsx.TileWidth), 0)
	}

	if tile.FlipFlag&tiled.FlipVertical != 0 {
		g.op.GeoM.Scale(1, -1)
		g.op.GeoM.Translate(0, float64(tsx.TileHeight))
	}

	g.op.GeoM.Translate(distX, distY)
	g.op.GeoM.Concat(g.camera.ViewMatrix())

	screen.DrawImage(img.SubImage(srcRect).(*ebiten.Image), &g.op)
}
