package spider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"WeMediaSpider/backend/pkg/logger"
)

// ImageInfo 图片信息
type ImageInfo struct {
	URL         string `json:"url"`
	Index       int    `json:"index"`
	Filename    string `json:"filename"`
	ArticleTitle string `json:"articleTitle"`
	AccountName string `json:"accountName"`
}

// ImageDownloadProgress 图片下载进度
type ImageDownloadProgress struct {
	Total     int    `json:"total"`
	Current   int    `json:"current"`
	Message   string `json:"message"`
	ImageURL  string `json:"imageUrl"`
	Success   bool   `json:"success"`
}

// ImageDownloader 图片下载器
type ImageDownloader struct {
	client  *http.Client
	headers map[string]string
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
}

// NewImageDownloader 创建图片下载器
func NewImageDownloader(headers map[string]string) *ImageDownloader {
	ctx, cancel := context.WithCancel(context.Background())
	return &ImageDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: headers,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Cancel 取消下载
func (d *ImageDownloader) Cancel() {
	d.cancel()
}

// ExtractImages 从文章内容中提取图片URL
func (d *ImageDownloader) ExtractImages(content string) []ImageInfo {
	var images []ImageInfo

	// 匹配 Markdown 格式的图片: ![alt](url)
	mdPattern := regexp.MustCompile(`!\[.*?\]\((https?://[^\)]+)\)`)
	mdMatches := mdPattern.FindAllStringSubmatch(content, -1)

	for _, match := range mdMatches {
		if len(match) > 1 {
			images = append(images, ImageInfo{
				URL:   match[1],
				Index: len(images) + 1,
			})
		}
	}

	// 匹配 HTML img 标签: <img src="url">
	htmlPattern := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
	htmlMatches := htmlPattern.FindAllStringSubmatch(content, -1)

	for _, match := range htmlMatches {
		if len(match) > 1 {
			url := match[1]
			// 去重
			exists := false
			for _, img := range images {
				if img.URL == url {
					exists = true
					break
				}
			}
			if !exists {
				images = append(images, ImageInfo{
					URL:   url,
					Index: len(images) + 1,
				})
			}
		}
	}

	// 重新编号
	for i := range images {
		images[i].Index = i + 1
		images[i].Filename = fmt.Sprintf("%d%s", i+1, d.getExtension(images[i].URL))
	}

	logger.Infof("从内容中提取到 %d 张图片", len(images))
	return images
}

// DownloadImagesWithProgress 下载图片并报告进度
func (d *ImageDownloader) DownloadImagesWithProgress(
	images []ImageInfo,
	baseDir string,
	maxWorkers int,
	progressChan chan<- ImageDownloadProgress,
) error {
	defer close(progressChan)

	if len(images) == 0 {
		return nil
	}

	logger.Infof("开始下载 %d 张图片,并发数: %d", len(images), maxWorkers)

	// 按公众号和文章标题分组
	type articleKey struct {
		accountName  string
		articleTitle string
	}
	articleGroups := make(map[articleKey][]ImageInfo)
	for _, img := range images {
		key := articleKey{
			accountName:  img.AccountName,
			articleTitle: img.ArticleTitle,
		}
		articleGroups[key] = append(articleGroups[key], img)
	}

	// 创建工作池
	type downloadTask struct {
		image     ImageInfo
		outputDir string
		index     int
	}

	tasks := make(chan downloadTask, len(images))
	results := make(chan ImageDownloadProgress, len(images))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				select {
				case <-d.ctx.Done():
					return
				default:
				}

				// 下载图片
				err := d.downloadImage(task.image.URL, filepath.Join(task.outputDir, task.image.Filename))

				progress := ImageDownloadProgress{
					Total:    len(images),
					Current:  task.index + 1,
					ImageURL: task.image.URL,
					Success:  err == nil,
				}

				if err != nil {
					progress.Message = fmt.Sprintf("下载失败: %s", task.image.Filename)
					logger.Errorf("下载图片失败 %s: %v", task.image.URL, err)
				} else {
					progress.Message = fmt.Sprintf("已下载: %s/%s/%s", task.image.AccountName, task.image.ArticleTitle, task.image.Filename)
				}

				results <- progress

				// 添加延迟，避免请求过快
				time.Sleep(200 * time.Millisecond)
			}
		}()
	}

	// 发送任务
	go func() {
		taskIndex := 0
		for key, articleImages := range articleGroups {
			// 创建 公众号/文章标题 目录
			accountDir := filepath.Join(baseDir, sanitizeFolderName(key.accountName))
			articleDir := filepath.Join(accountDir, sanitizeFolderName(key.articleTitle))

			if err := os.MkdirAll(articleDir, 0755); err != nil {
				logger.Errorf("创建目录失败 %s: %v", articleDir, err)
				continue
			}

			// 为每张图片创建任务
			for _, img := range articleImages {
				tasks <- downloadTask{
					image:     img,
					outputDir: articleDir,
					index:     taskIndex,
				}
				taskIndex++
			}
		}
		close(tasks)
	}()

	// 收集结果并发送进度
	go func() {
		wg.Wait()
		close(results)
	}()

	// 转发进度
	for progress := range results {
		select {
		case <-d.ctx.Done():
			return d.ctx.Err()
		case progressChan <- progress:
		}
	}

	logger.Infof("图片下载完成")
	return nil
}

// downloadImage 下载单张图片
func (d *ImageDownloader) downloadImage(url, filepath string) error {
	// 创建请求
	req, err := http.NewRequestWithContext(d.ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// 设置请求头
	for key, value := range d.headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入文件
	_, err = io.Copy(file, resp.Body)
	return err
}

// getExtension 从URL获取文件扩展名
func (d *ImageDownloader) getExtension(url string) string {
	// 移除查询参数
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	// 获取扩展名
	ext := filepath.Ext(url)
	if ext == "" {
		// 尝试从URL中识别格式
		if strings.Contains(url, "wx_fmt=jpeg") || strings.Contains(url, ".jpg") {
			return ".jpg"
		} else if strings.Contains(url, "wx_fmt=png") || strings.Contains(url, ".png") {
			return ".png"
		} else if strings.Contains(url, "wx_fmt=gif") || strings.Contains(url, ".gif") {
			return ".gif"
		}
		return ".jpg" // 默认
	}

	return ext
}

// sanitizeFolderName 清理文件夹名称
func sanitizeFolderName(name string) string {
	// 替换不允许的字符
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	return replacer.Replace(name)
}
