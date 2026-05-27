const fs = require("node:fs");
const fsp = require("node:fs/promises");
const http = require("node:http");
const path = require("node:path");
const puppeteer = require("puppeteer");
const Jimp = require("jimp");
const jsQR = require("jsqr");
const qrcode = require("qrcode-terminal");

function resolveRuntimeDir() {
  const candidates = [];

  if (process.env.DOUYU_SCAN_RUNTIME_DIR) {
    candidates.push(process.env.DOUYU_SCAN_RUNTIME_DIR);
  }

  candidates.push("/app/runtime");
  candidates.push(path.join(path.resolve(__dirname, "..", ".."), "runtime"));

  for (const candidate of candidates) {
    if (candidate && fs.existsSync(candidate)) {
      return candidate;
    }
  }

  return candidates[0];
}

const runtimeDir = resolveRuntimeDir();
const outputPath = path.join(runtimeDir, "stream.env");
const qrOutputPath = path.join(runtimeDir, "douyu-login-qr.png");
const roomUrl = process.env.DOUYU_SCAN_ROOM_URL || "https://www.douyu.com/creator/main/live";
const notifyEventURL = process.env.DOUYU_APP_NOTIFY_EVENT_URL || "http://app:8080/notify/event";
const browserPath =
  process.env.DOUYU_SCAN_BROWSER_PATH || "/usr/bin/chromium" || "/usr/bin/chromium-browser";
const headless = String(process.env.DOUYU_SCAN_HEADLESS || "true").toLowerCase() !== "false";
const blockedResourceTypes = new Set(["image", "font", "media", "manifest"]);
const blockedURLPatterns = [
  "googlesyndication",
  "doubleclick.net",
  "google-analytics.com",
  "googletagmanager.com",
  "googleadservices.com",
  "adsystem",
  "hm.baidu.com",
  "cnzz.com",
  "umeng.com",
  "sentry",
  "slardar",
  "metrics",
  "tracker",
  "beacon"
];

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForURLContains(page, expected, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const currentURL = page.url();
      if (currentURL && currentURL.includes(expected)) {
        return;
      }
    } catch (error) {
      // Ignore transient navigation/context errors while the page is changing.
    }
    await sleep(1000);
  }
  throw new Error(`timed out waiting for URL containing: ${expected}`);
}

async function decodeQrFromBuffer(buffer) {
  const image = await Jimp.read(buffer);
  const { data, width, height } = image.bitmap;
  const code = jsQR(new Uint8ClampedArray(data), width, height);
  return code && code.data ? code.data : null;
}

async function writeRuntimeEnv(rtmpUrl, streamKey) {
  const lines = [
    `DOUYU_STREAMER_STREAM_RTMP_URL=${rtmpUrl}`,
    `DOUYU_STREAMER_STREAM_KEY=${streamKey}`,
    `DOUYU_RELAY_FORWARD_URL=${rtmpUrl.replace(/\/+$/, "")}/${streamKey}`
  ];
  await fsp.writeFile(outputPath, `${lines.join("\n")}\n`, "utf8");
  console.log(`wrote runtime stream env: ${outputPath}`);
}

async function sendScanSuccessEmail() {
  const payload = JSON.stringify({
    summary: "扫码成功",
    detail: "斗鱼扫码成功，已获取新的推流地址和推流码。"
  });

  await new Promise((resolve, reject) => {
    const request = http.request(
      notifyEventURL,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Content-Length": Buffer.byteLength(payload)
        },
        timeout: 15000
      },
      (response) => {
        let body = "";
        response.setEncoding("utf8");
        response.on("data", (chunk) => {
          body += chunk;
        });
        response.on("end", () => {
          if (response.statusCode && response.statusCode >= 200 && response.statusCode < 300) {
            resolve();
            return;
          }
          reject(new Error(`notify event failed: status=${response.statusCode} body=${body}`));
        });
      }
    );

    request.on("timeout", () => {
      request.destroy(new Error("notify event timeout"));
    });
    request.on("error", reject);
    request.write(payload);
    request.end();
  });

  console.log("scan success email sent");
}

function isValidDouyuRTMPURL(value) {
  if (!value) {
    return false;
  }
  return /^rtmp:\/\/send[a-z0-9.-]*\.douyu\.com\/live$/i.test(value.trim());
}

function isValidDouyuStreamKey(value) {
  if (!value) {
    return false;
  }

  const trimmed = value.trim();
  if (trimmed === "replace-me" || trimmed.includes("send.example.douyu.com")) {
    return false;
  }

  return /wsSecret=/.test(trimmed) && /wsTime=/.test(trimmed) && trimmed.length >= 32;
}

async function readClipboardText(page) {
  return page.evaluate(async () => {
    if (!navigator.clipboard || !navigator.clipboard.readText) {
      return "";
    }
    try {
      return await navigator.clipboard.readText();
    } catch (error) {
      return "";
    }
  });
}

async function clickXPathIfPresent(page, xpath, timeout) {
  try {
    await page.waitForXPath(xpath, { timeout });
    const nodes = await page.$x(xpath);
    if (nodes[0]) {
      await nodes[0].click();
      return true;
    }
  } catch (error) {
    return false;
  }
  return false;
}

async function run() {
  await fsp.mkdir(runtimeDir, { recursive: true });

  const launchOptions = {
    headless: headless ? "new" : false,
    executablePath: fs.existsSync(browserPath) ? browserPath : undefined,
    protocolTimeout: 600000,
    args: [
      "--no-sandbox",
      "--disable-setuid-sandbox",
      "--disable-dev-shm-usage",
      "--start-maximized"
    ],
    defaultViewport: { width: 1920, height: 1080 }
  };

  const browser = await puppeteer.launch(launchOptions);

  try {
    const page = await browser.newPage();

    const context = browser.defaultBrowserContext();
    await context.overridePermissions("https://www.douyu.com", ["clipboard-write", "clipboard-read"]);

    await page.setRequestInterception(true);
    page.on("request", (request) => {
      const resourceType = request.resourceType();
      const requestURL = request.url().toLowerCase();
      const shouldBlockByType = blockedResourceTypes.has(resourceType);
      const shouldBlockByURL = blockedURLPatterns.some((pattern) => requestURL.includes(pattern));

      if (shouldBlockByType || shouldBlockByURL) {
        request.abort().catch(() => {});
        return;
      }
      request.continue().catch(() => {});
    });

    await page.goto("https://passport.douyu.com/", {
      waitUntil: "domcontentloaded",
      timeout: 0
    });

    await page.waitForXPath("//canvas", { timeout: 0 });
    const canvasNodes = await page.$x("//canvas");
    if (!canvasNodes[0]) {
      throw new Error("Canvas element not found.");
    }

    const qrDataUrl = await page.evaluate((canvas) => canvas.toDataURL("image/png"), canvasNodes[0]);
    const [, encoded] = qrDataUrl.split(",", 2);
    const qrBuffer = Buffer.from(encoded, "base64");
    await fsp.writeFile(qrOutputPath, qrBuffer);
    console.log(`wrote qr image: ${qrOutputPath}`);

    const qrText = await decodeQrFromBuffer(qrBuffer);
    if (qrText) {
      qrcode.generate(qrText, { small: true });
    } else {
      console.log("qr decode failed, use the saved image instead");
    }

    await waitForURLContains(page, "https://www.douyu.com/", 300000);

    console.log("scan complete, opening creator live page");
    await page.goto(roomUrl, { waitUntil: "domcontentloaded", timeout: 0 });

    await clickXPathIfPresent(page, "/html/body/div[4]/div/div/div/div/div/div/div[2]", 5000);

    for (let i = 0; i < 4; i += 1) {
      const ok = await clickXPathIfPresent(page, '//*[@id="root"]/div[4]/div[2]/div[2]', 5000);
      if (!ok) {
        break;
      }
    }

    const clickedStart = await clickXPathIfPresent(
      page,
      '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/span[1]',
      20000
    );
    if (clickedStart) {
      await sleep(1000);
      await clickXPathIfPresent(page, "/html/body/div[4]/div/div[2]/div/div/div/div[3]/div[2]", 20000);
    }

    await page.waitForXPath('//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]', {
      timeout: 0
    });

    await page.evaluate(() => {
      const node = document.evaluate(
        '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]',
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
      ).singleNodeValue;
      const button = node && node.lastChild;
      if (button) {
        button.dispatchEvent(
          new MouseEvent("click", {
            bubbles: true,
            cancelable: true,
            view: window
          })
        );
      }
    });

    await sleep(1000);
    const rtmpUrl = await readClipboardText(page);

    await page.evaluate(() => {
      const node = document.evaluate(
        '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[2]',
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
      ).singleNodeValue;
      const button = node && node.lastChild;
      if (button) {
        button.dispatchEvent(
          new MouseEvent("click", {
            bubbles: true,
            cancelable: true,
            view: window
          })
        );
      }
    });

    await sleep(1000);
    const streamKey = await readClipboardText(page);

    if (!rtmpUrl || !streamKey) {
      throw new Error("failed to fetch rtmp_url or stream_key from Douyu page");
    }
    if (!isValidDouyuRTMPURL(rtmpUrl)) {
      throw new Error(`invalid douyu rtmp_url: ${rtmpUrl}`);
    }
    if (!isValidDouyuStreamKey(streamKey)) {
      throw new Error(`invalid douyu stream_key: ${streamKey}`);
    }

    console.log(`rtmp_url: ${rtmpUrl}`);
    console.log(`stream_key: ${streamKey}`);
    await writeRuntimeEnv(rtmpUrl, streamKey);
    await sendScanSuccessEmail();
  } finally {
    await browser.close();
  }
}

run().catch((error) => {
  console.error(error);
  process.exit(1);
});
