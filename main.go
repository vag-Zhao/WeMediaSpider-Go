package main

import (
	"context"
	"embed"
	"flag"
	"os"

	"WeMediaSpider/backend/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 解析命令行参数
	silent := flag.Bool("silent", false, "Start in silent mode (hidden to tray)")
	flag.Parse()

	// Create an instance of the app structure
	application := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "WeMediaSpider - 微信公众号爬虫",
		Width:     1100,
		Height:    700,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 26, A: 1},
		OnStartup: func(ctx context.Context) {
			application.Startup(ctx)

			// 如果是静默启动，隐藏窗口
			if *silent {
				runtime.WindowHide(ctx)
			}
		},
		OnShutdown: application.Shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			// 检查是否应该阻止关闭
			return application.ShouldBlockClose()
		},
		Bind: []interface{}{
			application,
		},
		Frameless: true, // 隐藏系统标题栏
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
