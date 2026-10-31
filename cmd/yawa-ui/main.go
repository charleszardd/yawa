package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/config"
	"github.com/charleszardd/yawa/internal/dialogs"
	"github.com/charleszardd/yawa/internal/launcher"
	"github.com/charleszardd/yawa/internal/ui"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	windowsoptions "github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	configPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	windowsLauncher := launcher.Windows{}
	service := ui.NewService(configPath, command.NewDispatcher(windowsLauncher, dialogs.Picker{}))
	cacheDirectory, err := os.UserCacheDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	webviewDataPath := filepath.Join(cacheDirectory, "yawa", "webview2")
	frontendAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = wails.Run(&options.App{
		Title: "yawa", Width: 1120, Height: 760, MinWidth: 900, MinHeight: 620,
		AssetServer:      &assetserver.Options{Assets: frontendAssets},
		BackgroundColour: &options.RGBA{R: 12, G: 15, B: 22, A: 255},
		Bind:             []interface{}{service},
		Debug:            options.Debug{OpenInspectorOnStartup: os.Getenv("YAWA_DEBUG") == "1"},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "4f9ca656-9425-4c03-a53a-f72368391be1",
		},
		Windows: &windowsoptions.Options{
			WebviewUserDataPath: webviewDataPath,
			Theme:               windowsoptions.Dark,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
