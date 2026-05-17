package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	cdruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/liyue201/goqr"
	"github.com/makiuchi-d/gozxing"
	gozxingqrcode "github.com/makiuchi-d/gozxing/qrcode"
	"github.com/mdp/qrterminal/v3"
)

const (
	loginURL           = "https://passport.douyu.com/"
	defaultRoomURL     = "https://www.douyu.com/creator/main/live"
	pageReadyTimeout   = 30 * time.Second
	loginWaitTimeout   = 5 * time.Minute
	interactionTimeout = 20 * time.Second
	browserTimeout     = 30 * time.Minute
	pollInterval       = 500 * time.Millisecond

	dismissDialogXPath = `/html/body/div[4]/div/div/div/div/div/div/div[2]`
	repeatButtonXPath  = `//*[@id="root"]/div[4]/div[2]/div[2]`
	revealButtonXPath  = `//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/span[1]`
	confirmRevealXPath = `/html/body/div[4]/div/div[2]/div/div/div/div[3]/div[2]`
	rtmpURLXPath       = `//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]`
	streamKeyXPath     = `//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[2]`
)

var (
	rtmpURLPattern   = regexp.MustCompile(`rtmp://[^\s]+`)
	streamKeyPattern = regexp.MustCompile(`[A-Za-z0-9_-]{8,}`)
)

func main() {
	log.SetOutput(os.Stdout)

	rootDir, err := repoRoot()
	if err != nil {
		log.Fatal(err)
	}

	runtimeDir := filepath.Join(rootDir, "runtime")
	outputPath := filepath.Join(runtimeDir, "stream.env")
	qrOutputPath := filepath.Join(runtimeDir, "douyu-login-qr.png")

	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		log.Fatalf("create runtime dir: %v", err)
	}

	roomURL := envOrDefault("DOUYU_SCAN_ROOM_URL", defaultRoomURL)
	browserPath := os.Getenv("DOUYU_SCAN_BROWSER_PATH")
	headless := strings.ToLower(envOrDefault("DOUYU_SCAN_HEADLESS", "true")) != "false"

	ctx, cancel, err := newBrowserContext(browserPath, headless)
	if err != nil {
		log.Fatalf("create browser context: %v", err)
	}
	defer cancel()

	if err := chromedp.Run(ctx, browser.GrantPermissions(
		[]browser.PermissionType{
			browser.PermissionTypeClipboardReadWrite,
			browser.PermissionTypeClipboardSanitizedWrite,
		},
	).WithOrigin("https://www.douyu.com")); err != nil {
		log.Fatalf("grant clipboard permissions: %v", err)
	}

	log.Println("opening Douyu login page")
	if err := navigateAndWait(ctx, loginURL); err != nil {
		log.Fatalf("open login page: %v", err)
	}

	if err := waitForJSCondition(ctx, `(() => {
		const node = document.querySelector("canvas");
		if (!node) return false;
		const rect = node.getBoundingClientRect();
		return rect.width > 0 && rect.height > 0;
	})()`, pageReadyTimeout); err != nil {
		log.Fatalf("wait for QR canvas: %v", err)
	}

	qrDataURL, err := evalString(ctx, `(() => {
		const node = document.querySelector("canvas");
		return node ? node.toDataURL("image/png") : "";
	})()`)
	if err != nil {
		log.Fatalf("read QR canvas: %v", err)
	}

	qrBuffer, err := decodeDataURL(qrDataURL)
	if err != nil {
		log.Fatalf("decode QR image: %v", err)
	}

	if err := os.WriteFile(qrOutputPath, qrBuffer, 0o644); err != nil {
		log.Fatalf("write QR image: %v", err)
	}
	log.Printf("wrote qr image: %s", qrOutputPath)

	qrText, err := decodeQRCodeInBrowser(ctx)
	if err != nil || qrText == "" {
		qrText, err = decodeQRCodeWithGoZXing(qrBuffer)
	}
	if err != nil || qrText == "" {
		qrText, err = decodeQRCode(qrBuffer)
	}
	if err != nil {
		log.Printf("qr decode failed, rendering image directly in terminal: %v", err)
		if renderErr := printTerminalQRImage(qrBuffer); renderErr != nil {
			log.Printf("terminal QR image render failed, use the saved image instead: %v", renderErr)
		}
	} else {
		rendered, renderErr := renderTerminalQRCode(qrText)
		if renderErr != nil {
			log.Printf("failed to render terminal QR from text: %v", renderErr)
		} else {
			fmt.Printf("scan this QR code:\n%s\n", rendered)
		}
	}

	log.Printf("waiting for QR scan (headless=%t)", headless)
	if err := waitForJSCondition(ctx, `window.location.href.startsWith("https://www.douyu.com/")`, loginWaitTimeout); err != nil {
		log.Fatalf("wait for scan completion: %v", err)
	}

	log.Println("scan complete, opening creator live page")
	if err := navigateAndWait(ctx, roomURL); err != nil {
		log.Fatalf("open creator live page: %v", err)
	}

	_, _ = clickXPathIfVisible(ctx, dismissDialogXPath, 5*time.Second, false)

	for i := 0; i < 4; i++ {
		ok, err := clickXPathIfVisible(ctx, repeatButtonXPath, 3*time.Second, false)
		if err != nil || !ok {
			break
		}
	}

	if ok, _ := clickXPathIfVisible(ctx, revealButtonXPath, 5*time.Second, false); ok {
		_, _ = clickXPathIfVisible(ctx, confirmRevealXPath, 5*time.Second, false)
	}

	if err := waitForXPathVisible(ctx, rtmpURLXPath, interactionTimeout); err != nil {
		log.Fatalf("wait for push stream panel: %v", err)
	}

	log.Println("reading rtmp url")
	rtmpRaw, err := extractValue(ctx, "rtmp_url", rtmpURLXPath, false)
	if err != nil {
		log.Fatalf("read rtmp url: %v", err)
	}

	log.Println("reading stream key")
	streamKeyRaw, err := extractValue(ctx, "stream_key", streamKeyXPath, true)
	if err != nil {
		log.Fatalf("read stream key: %v", err)
	}

	rtmpURL := normalizeRTMPURL(rtmpRaw)
	streamKey := normalizeStreamKey(streamKeyRaw)
	if rtmpURL == "" || streamKey == "" {
		log.Fatal("failed to fetch rtmp_url or stream_key from Douyu page")
	}

	log.Printf("rtmp_url: %s", rtmpURL)
	log.Printf("stream_key: %s", streamKey)
	if err := writeRuntimeEnv(outputPath, rtmpURL, streamKey); err != nil {
		log.Fatalf("write runtime env: %v", err)
	}
}

func repoRoot() (string, error) {
	if root := os.Getenv("DOUYU_SCAN_ROOT_DIR"); root != "" {
		return root, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("failed to locate repository root from %s", wd)
		}
		current = parent
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func newBrowserContext(browserPath string, headless bool) (context.Context, context.CancelFunc, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-domain-reliability", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "AudioServiceOutOfProcess,VizDisplayCompositor"),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
		chromedp.WindowSize(1440, 960),
	)
	if browserPath != "" {
		opts = append(opts, chromedp.ExecPath(browserPath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	ctx, timeoutCancel := context.WithTimeout(taskCtx, browserTimeout)

	cancel := func() {
		timeoutCancel()
		taskCancel()
		allocCancel()
	}
	return ctx, cancel, nil
}

func navigateAndWait(ctx context.Context, url string) error {
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		return err
	}
	_ = chromedp.Run(ctx, chromedp.Evaluate(`(() => {
		window.__douyuLastCopiedText = "";
		if (!window.__douyuCopyHookInstalled) {
			const originalClipboard = navigator.clipboard && navigator.clipboard.writeText
				? navigator.clipboard.writeText.bind(navigator.clipboard)
				: null;
			if (originalClipboard) {
				navigator.clipboard.writeText = async (text) => {
					if (typeof text === "string" && text) {
						window.__douyuLastCopiedText = text;
					}
					return originalClipboard(text);
				};
			}

			document.addEventListener("copy", (event) => {
				try {
					const clipboardText = event.clipboardData && event.clipboardData.getData
						? event.clipboardData.getData("text/plain")
						: "";
					if (clipboardText) {
						window.__douyuLastCopiedText = clipboardText;
					}
				} catch {}
			}, true);
			window.__douyuCopyHookInstalled = true;
		}

		const style = document.createElement("style");
		style.textContent =
			"*, *::before, *::after {" +
			"animation: none !important;" +
			"transition: none !important;" +
			"scroll-behavior: auto !important;" +
			"}";
		document.documentElement.appendChild(style);
		return true;
	})()`, nil))
	return waitForJSCondition(ctx, `document.readyState !== "loading"`, pageReadyTimeout)
}

func waitForJSCondition(ctx context.Context, expression string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		var ok bool
		err := chromedp.Run(ctx, chromedp.Evaluate(expression, &ok))
		if err == nil && ok {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		time.Sleep(pollInterval)
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("timeout waiting for condition")
}

func waitForXPathVisible(ctx context.Context, xpath string, timeout time.Duration) error {
	js := fmt.Sprintf(`(() => {
		const node = document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!node) return false;
		const rect = node.getBoundingClientRect();
		return rect.width > 0 && rect.height > 0;
	})()`, jsString(xpath))
	return waitForJSCondition(ctx, js, timeout)
}

func clickXPathIfVisible(ctx context.Context, xpath string, timeout time.Duration, clickLastChild bool) (bool, error) {
	if err := waitForXPathVisible(ctx, xpath, timeout); err != nil {
		return false, nil
	}

	js := fmt.Sprintf(`(() => {
		const node = document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!node) return false;
		node.scrollIntoView({block: "center"});
		const target = %s ? node.lastElementChild : node;
		if (!target) return false;
		target.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
		return true;
	})()`, jsString(xpath), boolLiteral(clickLastChild))

	var clicked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &clicked)); err != nil {
		return false, err
	}
	return clicked, nil
}

func extractValue(ctx context.Context, fieldName, xpath string, preferClipboard bool) (string, error) {
	raw, err := evalString(ctx, fmt.Sprintf(`(() => {
		const node = document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!node) return "";
		const texts = [];
		const pushText = (value) => {
			if (typeof value !== "string") return;
			const trimmed = value.trim();
			if (trimmed) texts.push(trimmed);
		};
		pushText(node.value);
		pushText(node.getAttribute && node.getAttribute("value"));
		pushText(node.dataset && node.dataset.clipboardText);
		pushText(node.dataset && node.dataset.copyText);

		const walker = document.createTreeWalker(node, NodeFilter.SHOW_ELEMENT);
		for (let current = walker.currentNode; current; current = walker.nextNode()) {
			pushText(current.value);
			pushText(current.getAttribute && current.getAttribute("value"));
			pushText(current.getAttribute && current.getAttribute("data-clipboard-text"));
			pushText(current.getAttribute && current.getAttribute("data-copy-text"));
			pushText(current.getAttribute && current.getAttribute("title"));
		}

		for (const text of texts) {
			if (text.includes("rtmp://") || /[A-Za-z0-9_-]{8,}/.test(text)) {
				return text;
			}
		}

		const text = node.innerText || node.textContent || "";
		return text.trim();
	})()`, jsString(xpath)))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(raw) != "" {
		log.Printf("%s raw candidate: %s", fieldName, raw)
	} else {
		log.Printf("%s raw candidate is empty", fieldName)
	}
	if !preferClipboard && raw != "" && raw != "rtmp地址" && raw != "直播码" && !strings.Contains(raw, "********************************") {
		log.Printf("%s using raw value without clipboard", fieldName)
		return raw, nil
	}
	if preferClipboard {
		if fieldName == "stream_key" {
			log.Printf("%s invoking svg onClick handler directly", fieldName)
			callCtx, callCancel := context.WithTimeout(ctx, 2*time.Second)
			invoked, invokeErr := invokeSVGClickHandler(callCtx, xpath)
			callCancel()
			if invokeErr != nil {
				log.Printf("%s invoke svg onClick failed: %v", fieldName, invokeErr)
			} else {
				log.Printf("%s invoke svg onClick done (invoked=%t)", fieldName, invoked)
			}
		} else {
			log.Printf("%s clicking last child for clipboard", fieldName)
			clickCtx, clickCancel := context.WithTimeout(ctx, 2*time.Second)
			clicked, clickErr := clickXPathLastChild(clickCtx, xpath)
			clickCancel()
			if clickErr != nil {
				log.Printf("%s click last child failed: %v", fieldName, clickErr)
				return "", clickErr
			}
			log.Printf("%s clicking last child done (clicked=%t)", fieldName, clicked)
		}

		time.Sleep(1 * time.Second)
		clipboard, clipErr := evalStringAwaitPromise(ctx, `(async () => {
			if (!navigator.clipboard || !navigator.clipboard.readText) return "";
			try {
				return await navigator.clipboard.readText();
			} catch {
				return "";
			}
		})()`)
		if clipErr != nil {
			return "", clipErr
		}
		if strings.TrimSpace(clipboard) != "" {
			log.Printf("%s using clipboard value", fieldName)
			return clipboard, nil
		}
		log.Printf("%s clipboard is empty; falling back to raw candidate", fieldName)
	}

	if raw != "" && raw != "rtmp地址" && raw != "直播码" {
		log.Printf("%s using raw value", fieldName)
		return raw, nil
	}
	return "", nil
}

func clickXPathLastChild(ctx context.Context, xpath string) (bool, error) {
	js := fmt.Sprintf(`(() => {
		const node = document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!node || !node.lastChild) return false;
		node.lastChild.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
		return true;
	})()`, jsString(xpath))

	var clicked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &clicked)); err != nil {
	return false, err
	}
	return clicked, nil
}

func invokeSVGClickHandler(ctx context.Context, xpath string) (bool, error) {
	js := fmt.Sprintf(`(() => {
		const root = document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!root) return false;
		const svg = root.querySelector("svg");
		if (!svg) return false;

		const keys = Object.keys(svg);
		for (const key of keys) {
			if (!key.startsWith("__reactProps$")) continue;
			const props = svg[key];
			if (props && typeof props.onClick === "function") {
				const nativeEvent = new MouseEvent("click", {
					bubbles: true,
					cancelable: true,
					view: window,
					button: 0,
					buttons: 1,
				});
				Object.defineProperty(nativeEvent, "target", { value: svg, configurable: true });
				Object.defineProperty(nativeEvent, "currentTarget", { value: svg, configurable: true });

				let defaultPrevented = false;
				let propagationStopped = false;
				const event = {
					type: "click",
					target: svg,
					currentTarget: svg,
					nativeEvent,
					bubbles: true,
					cancelable: true,
					defaultPrevented: false,
					eventPhase: 3,
					isTrusted: true,
					timeStamp: Date.now(),
					button: 0,
					buttons: 1,
					clientX: 0,
					clientY: 0,
					pageX: 0,
					pageY: 0,
					screenX: 0,
					screenY: 0,
					metaKey: false,
					ctrlKey: false,
					shiftKey: false,
					altKey: false,
					persist() {},
					preventDefault() {
						defaultPrevented = true;
						this.defaultPrevented = true;
						if (this.nativeEvent && this.nativeEvent.preventDefault) {
							this.nativeEvent.preventDefault();
						}
					},
					isDefaultPrevented() {
						return defaultPrevented;
					},
					stopPropagation() {
						propagationStopped = true;
						if (this.nativeEvent && this.nativeEvent.stopPropagation) {
							this.nativeEvent.stopPropagation();
						}
					},
					isPropagationStopped() {
						return propagationStopped;
					},
				};
				props.onClick(event);
				return true;
			}
		}
		return false;
	})()`, jsString(xpath))

	var invoked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &invoked)); err != nil {
		return false, err
	}
	return invoked, nil
}

func evalString(ctx context.Context, expression string) (string, error) {
	var result string
	if err := chromedp.Run(ctx, chromedp.Evaluate(expression, &result)); err != nil {
		return "", err
	}
	return result, nil
}

func evalStringAwaitPromise(ctx context.Context, expression string) (string, error) {
	var result string
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		res, exp, err := cdruntime.Evaluate(expression).
			WithAwaitPromise(true).
			WithReturnByValue(true).
			Do(ctx)
		if err != nil {
			return err
		}
		if exp != nil {
			if exp.Exception != nil && exp.Exception.Description != "" {
				return fmt.Errorf("javascript exception: %s", exp.Exception.Description)
			}
			return fmt.Errorf("javascript exception: %s", exp.Error())
		}
		if res == nil || res.Value == nil {
			result = ""
			return nil
		}
		if err := json.Unmarshal(res.Value, &result); err == nil {
			return nil
		}
		result = string(res.Value)
		return nil
	})); err != nil {
		return "", err
	}
	return result, nil
}

func decodeQRCodeInBrowser(ctx context.Context) (string, error) {
	return evalString(ctx, `(() => {
		const node = document.querySelector("canvas");
		if (!node) return Promise.resolve("");
		if (typeof BarcodeDetector === "undefined") return Promise.resolve("");
		return (async () => {
			try {
				const detector = new BarcodeDetector({ formats: ["qr_code"] });
				const results = await detector.detect(node);
				if (!results || results.length === 0) return "";
				return results[0].rawValue || "";
			} catch {
				return "";
			}
		})()
	})()`)
}

func decodeDataURL(value string) ([]byte, error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data url")
	}
	return base64.StdEncoding.DecodeString(parts[1])
}

func decodeQRCode(buffer []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(buffer))
	if err != nil {
		return "", err
	}

	candidates := []image.Image{
		img,
		binarizeImage(img),
		resizeNearest(binarizeImage(img), 2),
		resizeNearest(binarizeImage(img), 3),
	}

	for _, candidate := range candidates {
		symbols, recErr := goqr.Recognize(candidate)
		if recErr != nil || len(symbols) == 0 {
			continue
		}
		return string(symbols[0].Payload), nil
	}

	return "", fmt.Errorf("no qr code detected")
}

func decodeQRCodeWithGoZXing(buffer []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(buffer))
	if err != nil {
		return "", err
	}

	candidates := []image.Image{
		img,
		binarizeImage(img),
		resizeNearest(binarizeImage(img), 2),
		resizeNearest(binarizeImage(img), 3),
	}

	reader := gozxingqrcode.NewQRCodeReader()
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}

	for _, candidate := range candidates {
		bmp, convErr := gozxing.NewBinaryBitmapFromImage(candidate)
		if convErr != nil {
			continue
		}
		result, decErr := reader.Decode(bmp, hints)
		if decErr != nil || result == nil {
			continue
		}
		text := strings.TrimSpace(result.GetText())
		if text != "" {
			return text, nil
		}
	}

	return "", fmt.Errorf("no qr code detected by gozxing")
}

func printTerminalQRImage(buffer []byte) error {
	rendered, err := renderTerminalQRImage(buffer)
	if err != nil {
		return err
	}
	fmt.Printf("scan this QR code:\n%s\n", rendered)
	return nil
}

func renderTerminalQRCode(text string) (string, error) {
	var out strings.Builder
	config := qrterminal.Config{
		Level:      qrterminal.M,
		Writer:     &out,
		HalfBlocks: true,
		QuietZone:  4,
	}
	qrterminal.GenerateWithConfig(text, config)
	return strings.TrimRight(out.String(), "\n"), nil
}

func renderTerminalQRImage(buffer []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(buffer))
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return "", fmt.Errorf("empty qr image")
	}

	const maxModules = 20
	scaleX := max(1, width/maxModules)
	scaleY := max(1, height/maxModules)
	scale := max(scaleX, scaleY)

	const cellWidth = 2
	const cellHeight = 4

	var out strings.Builder
	for y := bounds.Min.Y; y < bounds.Max.Y; y += scale * cellHeight {
		var line strings.Builder
		for x := bounds.Min.X; x < bounds.Max.X; x += scale * cellWidth {
			var bits rune
			for dx := 0; dx < cellWidth; dx++ {
				for dy := 0; dy < cellHeight; dy++ {
					px := min(x+dx*scale, bounds.Max.X-1)
					py := min(y+dy*scale, bounds.Max.Y-1)
					if !isDark(img.At(px, py)) {
						continue
					}
					switch {
					case dx == 0 && dy == 0:
						bits |= 1 << 0
					case dx == 0 && dy == 1:
						bits |= 1 << 1
					case dx == 0 && dy == 2:
						bits |= 1 << 2
					case dx == 0 && dy == 3:
						bits |= 1 << 6
					case dx == 1 && dy == 0:
						bits |= 1 << 3
					case dx == 1 && dy == 1:
						bits |= 1 << 4
					case dx == 1 && dy == 2:
						bits |= 1 << 5
					case dx == 1 && dy == 3:
						bits |= 1 << 7
					}
				}
			}
			if bits == 0 {
				line.WriteRune(' ')
			} else {
				line.WriteRune(rune(0x2800 + bits))
			}
		}
		out.WriteString(line.String())
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func binarizeImage(src image.Image) *image.Gray {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)

	var total uint64
	var count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			luma := uint8(((299*r + 587*g + 114*b) / 1000) >> 8)
			total += uint64(luma)
			count++
		}
	}

	threshold := uint8(128)
	if count > 0 {
		threshold = uint8(total / count)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			luma := uint8(((299*r + 587*g + 114*b) / 1000) >> 8)
			if luma < threshold {
				dst.SetGray(x, y, color.Gray{Y: 0})
			} else {
				dst.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}

	return dst
}

func resizeNearest(src image.Image, factor int) *image.Gray {
	if factor <= 1 {
		gray := image.NewGray(src.Bounds())
		draw.Draw(gray, src.Bounds(), src, src.Bounds().Min, draw.Src)
		return gray
	}

	bounds := src.Bounds()
	dst := image.NewGray(image.Rect(0, 0, bounds.Dx()*factor, bounds.Dy()*factor))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			sx := bounds.Min.X + x/factor
			sy := bounds.Min.Y + y/factor
			r, g, b, _ := src.At(sx, sy).RGBA()
			luma := uint8(((299*r + 587*g + 114*b) / 1000) >> 8)
			dst.SetGray(x, y, color.Gray{Y: luma})
		}
	}
	return dst
}

func isDark(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	luma := (299*r + 587*g + 114*b) / 1000
	return luma < 0x8000
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeRTMPURL(raw string) string {
	if match := rtmpURLPattern.FindString(raw); match != "" {
		return strings.TrimSpace(match)
	}
	return strings.TrimSpace(raw)
}

func normalizeStreamKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lines := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r'
	})

	for i := len(lines) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lines[i])
		if candidate == "" || strings.HasPrefix(candidate, "rtmp://") {
			continue
		}
		if streamKeyPattern.FindString(candidate) != "" {
			return candidate
		}
	}

	return raw
}

func writeRuntimeEnv(outputPath, rtmpURL, streamKey string) error {
	relayForwardURL := fmt.Sprintf("%s/%s", strings.TrimRight(rtmpURL, "/"), streamKey)
	lines := []string{
		fmt.Sprintf("DOUYU_STREAMER_STREAM_RTMP_URL=%s", rtmpURL),
		fmt.Sprintf("DOUYU_STREAMER_STREAM_KEY=%s", streamKey),
		fmt.Sprintf("DOUYU_RELAY_FORWARD_URL=%s", relayForwardURL),
	}
	if err := os.WriteFile(outputPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	log.Printf("wrote runtime stream env: %s", outputPath)
	return nil
}

func jsString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func boolLiteral(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
