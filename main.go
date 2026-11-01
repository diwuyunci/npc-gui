package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"npc-gui/internal/model"
	"npc-gui/internal/update"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The freshly downloaded binary runs as a helper before Wails initialises.
	if update.IsHelperInvocation(os.Args) {
		os.Exit(update.RunHelper(os.Args[1:]))
	}
	hidden := false
	for _, arg := range os.Args[1:] {
		if arg == "--hidden" {
			hidden = true
		}
	}

	app := NewApp()
	app.SetStartHidden(hidden)
	err := wails.Run(&options.App{
		Title:                    model.AppName,
		Width:                    1280,
		Height:                   820,
		MinWidth:                 1040,
		MinHeight:                680,
		StartHidden:              hidden,
		BackgroundColour:         &options.RGBA{R: 11, G: 15, B: 20, A: 1},
		OnStartup:                app.startup,
		OnShutdown:               app.shutdown,
		OnBeforeClose:            app.beforeClose,
		EnableDefaultContextMenu: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "npc-gui-desktop-single-instance",
			OnSecondInstanceLaunch: app.secondInstance,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
