package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()
	messages := windows.DefaultMessages()
	messages.DownloadPage = "MetaClean requires Microsoft Edge WebView2 Runtime. Press OK to open Microsoft's official download page, or Cancel to exit. Nothing will be installed automatically. Minimum version required: "
	messages.ContactAdmin = "MetaClean requires Microsoft Edge WebView2 Runtime. Install it from https://developer.microsoft.com/microsoft-edge/webview2/ and restart MetaClean."

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "MetaClean",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 22, B: 27, A: 255},
		OnStartup:        app.startup,
		Windows:          &windows.Options{Messages: messages},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
