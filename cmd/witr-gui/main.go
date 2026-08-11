package main

import (
	"embed"

	"github.com/pranshuparmar/witr/internal/gui"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed icon.png
var appIcon []byte

func main() {
	// Create an instance of the app structure
	app := gui.NewApp()

	// Create application with options (Standard window management without system tray)
	err := wails.Run(&options.App{
		Title:             "witr — Why is this running?",
		Width:             1280,
		Height:            800,
		MinWidth:          820,
		MinHeight:         520,
		WindowStartState:  options.Maximised, // Maximized by default on launch
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false, // Standard close exits application
		BackgroundColour:  &options.RGBA{R: 15, G: 23, B: 42, A: 255}, // Dark Slate #0f172a
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:      nil, // No file toolbar
		OnStartup: app.Startup,
		Bind:      []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			Theme:                windows.Dark,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:   windows.RGB(15, 23, 42),
				DarkModeTitleText:  windows.RGB(248, 250, 252),
				DarkModeBorder:     windows.RGB(51, 65, 85),
				LightModeTitleBar:  windows.RGB(255, 255, 255),
				LightModeTitleText: windows.RGB(15, 23, 42),
				LightModeBorder:    windows.RGB(226, 232, 240),
			},
		},
	})

	if err != nil {
		println("Error starting witr GUI:", err.Error())
	}
}
