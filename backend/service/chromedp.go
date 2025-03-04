package service

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

var (
	GrafanaURL      string
	GrafanaUserName string
	GrafanaPassword string

	ChromedpHeadless bool = true

	DefaultAllocContext       context.Context // specify the chrome settings
	DefaultAllocContextCancel context.CancelFunc

	DefaultChromeContext       context.Context // a chrome window, share cookies, cache, etc
	DefaultChromeContextCancel context.CancelFunc
)

func createAllocContext(headless bool) (context.Context, context.CancelFunc) {

	_headless := headless

	boolValue, err := strconv.ParseBool(GetEnv("HEADLESS", "true"))
	if err == nil {
		_headless = boolValue
	}
	logger.Printf("createAllocContext invoke , headless: %v", _headless)

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", _headless),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("enable-automation", false),
		//chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-extensions", true),
	)
	if globalConfig.ChromeDP == "" {
		return chromedp.NewExecAllocator(context.Background(), opts...)
	}
	fmt.Println("use remote ", globalConfig.ChromeDP)
	return chromedp.NewRemoteAllocator(context.Background(), globalConfig.ChromeDP)

}

func loginChrome(url, userName, password string) error {

	DefaultAllocContext, DefaultAllocContextCancel = createAllocContext(ChromedpHeadless)

	DefaultChromeContext, DefaultChromeContextCancel = chromedp.NewContext(DefaultAllocContext)
	defer DefaultChromeContextCancel()

	logger.Println("Grafana login process begaining......")

	err := chromedp.Run(DefaultChromeContext, loginGrafanaTasks(url, userName, password))

	if err != nil {
		return err
	}

	logger.Println("Grafana login  successfully")

	return nil

}

func loginGrafanaTasks(grfanaURL, username, password string) chromedp.Tasks {
	println("t006", grfanaURL)
	return chromedp.Tasks{
		chromedp.Navigate(fmt.Sprintf("%s/login", grfanaURL)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentLocation string
			if err := chromedp.Run(ctx,
				chromedp.WaitReady(`body`),
				chromedp.Location(&currentLocation),
			); err != nil {
				return err
			}
			locationBase := path.Base(currentLocation)
			if locationBase != "login" {
				logger.Println("already login, skip login")
				return nil
			}
			return chromedp.Run(ctx, chromedp.WaitVisible(`input[name='user']`),
				chromedp.SendKeys(`input[name='user']`, username),
				chromedp.SendKeys(`input[name='password']`, password),
				chromedp.Click(`button[type='submit']`),
				chromedp.WaitReady(`.page-dashboard`))
		}),
	}
}

func ScreenCaptureTasks(targetURLs []string, conn *MonitorConnection) (map[string]string, error) {
	urlToImageMap := make(map[string]string)

	defaultAllocContext, defaultAllocContextCancel := createAllocContext(false)
	defer defaultAllocContextCancel()

	timeContext, cancelFunc := context.WithTimeout(defaultAllocContext, time.Second*90)
	defer cancelFunc()

	defaultChromeContext, defaultChromeContextCancel := chromedp.NewContext(timeContext)
	defer defaultChromeContextCancel()

	fmt.Println("t003", conn, targetURLs[0])
	err := chromedp.Run(defaultChromeContext, loginGrafanaTasks(conn.URL, conn.Username, conn.Password))
	if err != nil {
		fmt.Println("", err.Error())
		return nil, fmt.Errorf("failed to capture screenshot for  %v", err)
	}

	for _, url := range targetURLs {
		var buf []byte
		tasks := chromedp.Tasks{
			chromedp.Navigate(url),
			chromedp.WaitVisible(`.page-dashboard`),
			//chromedp.WaitVisible(`div[aria-label='Panel loading bar']`),
			chromedp.Sleep(2 * time.Second),
			//chromedp.WaitVisible(`div[class="graph-panel graph-panel--legend-right"]`),
			chromedp.CaptureScreenshot(&buf),
		}

		err := chromedp.Run(defaultChromeContext, tasks)
		if err != nil {
			return nil, fmt.Errorf("failed to capture screenshot for URL %s: %v", url, err)
		}

		// Save the screenshot to a file
		imagePath, err := saveScreenshotToFile(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to save screenshot for URL %s: %v", url, err)
		}

		urlToImageMap[url] = imagePath
	}
	logger.Printf("ScreenCaptureTask completed , %+v", urlToImageMap)
	return urlToImageMap, nil
}

func saveScreenshotToFile(data []byte) (string, error) {
	// 将PNG数据解码为image.Image
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode PNG: %v", err)
	}

	// 获取当前时间
	now := time.Now()

	// 构建文件路径
	year := strconv.Itoa(now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())
	dirPath := filepath.Join("static", "screenshots", year, month, day)
	fileName := fmt.Sprintf("%s.jpg", uuid.New().String()) // 改为.jpg扩展名
	filePath := filepath.Join(dirPath, fileName)

	// 创建目录
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return "", err
	}

	// 创建输出文件
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// 将图像编码为JPEG格式
	opts := jpeg.Options{
		Quality: 90,
	}
	if err := jpeg.Encode(out, img, &opts); err != nil {
		return "", fmt.Errorf("failed to encode JPEG: %v", err)
	}

	return filePath, nil
}
