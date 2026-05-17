package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	cdruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	browserPath := os.Getenv("DOUYU_SCAN_BROWSER_PATH")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.WindowSize(1280, 720),
	)
	if browserPath != "" {
		opts = append(opts, chromedp.ExecPath(browserPath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	ctx, cancel := context.WithTimeout(taskCtx, 30*time.Second)
	defer cancel()

	chromedp.ListenTarget(ctx, func(ev any) {
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			values := make([]string, 0, len(e.Args))
			for _, arg := range e.Args {
				if arg.Value != nil {
					values = append(values, string(arg.Value))
				} else if arg.Description != "" {
					values = append(values, arg.Description)
				}
			}
			log.Printf("console.%s: %v", e.Type.String(), values)
		case *runtime.EventExceptionThrown:
			log.Printf("exception: %s", e.ExceptionDetails.Text)
			if e.ExceptionDetails.Exception != nil && e.ExceptionDetails.Exception.Description != "" {
				log.Printf("exception detail: %s", e.ExceptionDetails.Exception.Description)
			}
		}
	})

	html := `<!doctype html>
<html>
<body>
  <div id="root">
    <div class="itemWrap--2f8TJ3y">
      <span class="fieldTitle--2VcWT2c">直播码</span>
      <input id="streamKeyInput" readonly type="text" class="shark-Input input--2DEflcU"
        value="9163452rSlYGcFzT?dyPRI=0&noforward=1&origin=hw&record=flv&roirecognition=0&stemp_id=12898962&tw=0&wm=0&wsSecret=abcdef&wsSeek=off&wsTime=6a0956f3">
      <svg id="copySvg" class="svgIcon--2ypAR1M svg--2uID9Py" aria-hidden="true">
        <use id="copyUse" xlink:href="#copyNew"></use>
      </svg>
    </div>
  </div>
  <script>
    window.__copied = "";
    const text = document.getElementById("streamKeyInput").value;
    const svg = document.getElementById("copySvg");
    console.log("probe page ready");
    svg.addEventListener("click", async () => {
      console.log("svg click handler start", "trusted=" + event.isTrusted);
      window.__copied = text;
      try {
        await navigator.clipboard.writeText(text);
        console.log("clipboard write ok");
      } catch (err) {
        window.__copyErr = String(err);
        console.log("clipboard write failed", String(err));
      }
    });
  </script>
</body>
</html>`

	dataURL := "data:text/html;charset=utf-8," + html
	if err := chromedp.Run(ctx, chromedp.Navigate(dataURL)); err != nil {
		log.Fatalf("navigate: %v", err)
	}

	runProbe := func(name, js string) {
		_, _ = evalString(ctx, `(() => { window.__copied = ""; window.__copyErr = ""; return ""; })()`)

		clickCtx, clickCancel := context.WithTimeout(ctx, 2*time.Second)
		var clicked bool
		err := chromedp.Run(clickCtx, chromedp.Evaluate(js, &clicked))
		clickCancel()
		log.Printf("%s click finished clicked=%t err=%v", name, clicked, err)

		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
		copied, copiedErr := evalString(readCtx, `(() => window.__copied || "")()`)
		clip, clipErr := evalStringAwaitPromise(readCtx, `(async () => {
			if (!navigator.clipboard || !navigator.clipboard.readText) return "";
			try {
				return await navigator.clipboard.readText();
			} catch {
				return "";
			}
		})()`)
		copyErr, copyErrEval := evalString(readCtx, `(() => window.__copyErr || "")()`)
		readCancel()
		log.Printf("%s copied=%q copiedErr=%v clipboard=%q clipErr=%v copyErr=%q copyErrEval=%v",
			name, copied, copiedErr, clip, clipErr, copyErr, copyErrEval)
	}

	runProbe("lastChild-dispatch", `(() => {
		const node = document.querySelector(".itemWrap--2f8TJ3y");
		if (!node || !node.lastChild) return false;
		node.lastChild.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
		return true;
	})()`)

	runProbe("svg-click", `(() => {
		const node = document.getElementById("copySvg");
		if (!node) return false;
		node.click();
		return true;
	})()`)

	runProbe("svg-dispatch", `(() => {
		const node = document.getElementById("copySvg");
		if (!node) return false;
		node.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
		return true;
	})()`)

	runProbe("use-click", `(() => {
		const node = document.getElementById("copyUse");
		if (!node) return false;
		node.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
		return true;
	})()`)

	runProbe("svg-full-sequence", `(() => {
		const node = document.getElementById("copySvg");
		if (!node) return false;
		const events = [
			new MouseEvent("mouseover", { bubbles: true, cancelable: true, view: window }),
			new PointerEvent("pointerdown", { bubbles: true, cancelable: true, pointerId: 1, pointerType: "mouse", isPrimary: true }),
			new MouseEvent("mousedown", { bubbles: true, cancelable: true, view: window }),
			new PointerEvent("pointerup", { bubbles: true, cancelable: true, pointerId: 1, pointerType: "mouse", isPrimary: true }),
			new MouseEvent("mouseup", { bubbles: true, cancelable: true, view: window }),
			new MouseEvent("click", { bubbles: true, cancelable: true, view: window }),
		];
		for (const ev of events) node.dispatchEvent(ev);
		return true;
	})()`)

	runProbe("lastElementChild-full-sequence", `(() => {
		const node = document.querySelector(".itemWrap--2f8TJ3y");
		const target = node && node.lastElementChild;
		if (!target) return false;
		const events = [
			new MouseEvent("mouseover", { bubbles: true, cancelable: true, view: window }),
			new PointerEvent("pointerdown", { bubbles: true, cancelable: true, pointerId: 1, pointerType: "mouse", isPrimary: true }),
			new MouseEvent("mousedown", { bubbles: true, cancelable: true, view: window }),
			new PointerEvent("pointerup", { bubbles: true, cancelable: true, pointerId: 1, pointerType: "mouse", isPrimary: true }),
			new MouseEvent("mouseup", { bubbles: true, cancelable: true, view: window }),
			new MouseEvent("click", { bubbles: true, cancelable: true, view: window }),
		];
		for (const ev of events) target.dispatchEvent(ev);
		return true;
	})()`)

	runActionProbe := func(name string, action chromedp.Action) {
		log.Printf("%s starting", name)
		_, _ = evalString(ctx, `(() => { window.__copied = ""; window.__copyErr = ""; return ""; })()`)

		actionCtx, actionCancel := context.WithTimeout(ctx, 2*time.Second)
		err := chromedp.Run(actionCtx, action)
		actionCancel()
		log.Printf("%s action finished err=%v", name, err)

		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
		copied, copiedErr := evalString(readCtx, `(() => window.__copied || "")()`)
		clip, clipErr := evalStringAwaitPromise(readCtx, `(async () => {
			if (!navigator.clipboard || !navigator.clipboard.readText) return "";
			try {
				return await navigator.clipboard.readText();
			} catch {
				return "";
			}
		})()`)
		copyErr, copyErrEval := evalString(readCtx, `(() => window.__copyErr || "")()`)
		readCancel()
		log.Printf("%s copied=%q copiedErr=%v clipboard=%q clipErr=%v copyErr=%q copyErrEval=%v",
			name, copied, copiedErr, clip, clipErr, copyErr, copyErrEval)
	}

	log.Printf("all dispatch probes finished")
	runActionProbe("chromedp-click-svg", chromedp.Click(`#copySvg`, chromedp.ByQuery))
	log.Printf("all probes finished")
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
