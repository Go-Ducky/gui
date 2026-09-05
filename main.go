package main

import (
	"embed"
	"log"

	"github.com/go-ducky/gui/internal/guiservice"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	svc := guiservice.NewService()

	app := application.New(application.Options{
		Name:        "GoDucky GUI",
		Description: "GoDucky — AI coding agent, desktop edition",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "GoDucky",
		Width: 1200,
		Height: 780,
		MinWidth: 820,
		MinHeight: 600,
		// Start the UI centered, full brightness.
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			Theme: application.SystemDefault,
		},
		BackgroundColour: application.NewRGB(15, 17, 26),
		URL: "/",
	})

	// Auto-save the active chat when the window closes, matching the CLI's
	// "saves automatically when you quit" behaviour.
	window.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		_ = svc.SaveSession("")
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}