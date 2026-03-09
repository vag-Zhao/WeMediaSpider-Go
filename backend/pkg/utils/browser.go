package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BrowserInfo 浏览器信息
type BrowserInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DetectBrowsers 检测系统中可用的主流浏览器（Chrome 和 Edge）
func DetectBrowsers() []BrowserInfo {
	browsers := []BrowserInfo{}

	if runtime.GOOS == "windows" {
		// Windows 浏览器路径 - 优先 Chrome，其次 Edge
		paths := []struct {
			name string
			path string
		}{
			{"Google Chrome", `C:\Program Files\Google\Chrome\Application\chrome.exe`},
			{"Google Chrome (x86)", `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`},
			{"Microsoft Edge", `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`},
			{"Microsoft Edge", `C:\Program Files\Microsoft\Edge\Application\msedge.exe`},
		}

		for _, browser := range paths {
			if _, err := os.Stat(browser.path); err == nil {
				// 检查是否已经添加过（避免重复）
				exists := false
				for _, b := range browsers {
					if b.Path == browser.path {
						exists = true
						break
					}
				}

				if !exists {
					browsers = append(browsers, BrowserInfo{
						Name: browser.name,
						Path: browser.path,
					})
					fmt.Printf("[浏览器检测] 发现 %s: %s\n", browser.name, browser.path)
				}
			}
		}

		// 检查用户目录下的 Chrome
		if homeDir, err := os.UserHomeDir(); err == nil {
			localChrome := filepath.Join(homeDir, `AppData\Local\Google\Chrome\Application\chrome.exe`)
			if _, err := os.Stat(localChrome); err == nil {
				exists := false
				for _, b := range browsers {
					if b.Path == localChrome {
						exists = true
						break
					}
				}

				if !exists {
					browsers = append(browsers, BrowserInfo{
						Name: "Google Chrome",
						Path: localChrome,
					})
					fmt.Printf("[浏览器检测] 发现 Google Chrome (User): %s\n", localChrome)
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		// macOS 浏览器路径
		paths := []struct {
			name string
			path string
		}{
			{"Google Chrome", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
			{"Microsoft Edge", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
		}

		for _, browser := range paths {
			if _, err := os.Stat(browser.path); err == nil {
				browsers = append(browsers, BrowserInfo{
					Name: browser.name,
					Path: browser.path,
				})
				fmt.Printf("[浏览器检测] 发现 %s: %s\n", browser.name, browser.path)
			}
		}
	} else if runtime.GOOS == "linux" {
		// Linux 浏览器路径
		paths := []struct {
			name string
			cmd  string
		}{
			{"Google Chrome", "google-chrome"},
			{"Microsoft Edge", "microsoft-edge"},
		}

		for _, browser := range paths {
			if path, err := exec.LookPath(browser.cmd); err == nil {
				browsers = append(browsers, BrowserInfo{
					Name: browser.name,
					Path: path,
				})
				fmt.Printf("[浏览器检测] 发现 %s: %s\n", browser.name, path)
			}
		}
	}

	if len(browsers) == 0 {
		fmt.Println("[浏览器检测] 警告: 未检测到 Chrome 或 Edge 浏览器")
	} else {
		fmt.Printf("[浏览器检测] 将使用: %s\n", browsers[0].Name)
	}

	return browsers
}

// GetDefaultBrowser 获取默认浏览器路径（优先 Chrome，其次 Edge）
func GetDefaultBrowser() string {
	browsers := DetectBrowsers()
	if len(browsers) > 0 {
		return browsers[0].Path
	}
	return ""
}
