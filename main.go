package main

import (
	"embed"

	"github.com/1UPFR/1UP/internal/binutil"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed binaries
var embeddedBinaries embed.FS

func main() {
	binutil.Init(embeddedBinaries)

	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "1UP",
		Width:  1440,
		Height: 900,
		MinWidth: 1200,
		MinHeight: 750,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:   &options.RGBA{R: 13, G: 17, B: 23, A: 1},
		OnStartup:          app.startup,
		OnShutdown:         app.shutdown,
		EnableDefaultContextMenu: true,
		DragAndDrop:        &options.DragAndDrop{EnableFileDrop: true},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
