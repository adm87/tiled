module github.com/adm87/tiled/examples/profiling

go 1.25.2

replace github.com/adm87/tiled => ../../

replace github.com/adm87/tiled/examples/shared => ../shared

require github.com/adm87/tiled v0.1.3

require (
	github.com/adm87/enum v0.0.1 // indirect
	github.com/adm87/tiled/examples/shared v0.0.0-00010101000000-000000000000
	github.com/klauspost/compress v1.18.0 // indirect
)
