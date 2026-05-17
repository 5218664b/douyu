package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	browserPath := os.Getenv("DOUYU_SCAN_BROWSER_PATH")
	headless := true

	if browserPath != "" {
		cmd := exec.Command(
			browserPath,
			"--headless",
			"--no-sandbox",
			"--disable-gpu",
			"--disable-software-rasterizer",
			"--disable-features=VizDisplayCompositor",
			"--dump-dom",
			"https://example.com",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("chromium cli probe failed: %v", err)
			if len(output) > 0 {
				log.Printf("chromium cli probe output:\n%s", string(output))
			}
		} else {
			log.Printf("chromium cli probe ok (%d bytes)", len(output))
		}
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.WindowSize(1280, 720),
	)
	if browserPath != "" {
		opts = append(opts, chromedp.ExecPath(browserPath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	ctx, cancel := context.WithTimeout(taskCtx, 60*time.Second)
	defer cancel()

	var title string
	var href string

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://example.com"),
		chromedp.Title(&title),
		chromedp.Location(&href),
	); err != nil {
		log.Fatalf("smoke test failed: %v", err)
	}

	log.Printf("smoke test ok: title=%q href=%s", title, href)
}
