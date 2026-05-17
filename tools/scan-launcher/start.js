const fs = require("node:fs");
const fsp = require("node:fs/promises");
const path = require("node:path");
const puppeteer = require("puppeteer");
const Jimp = require("jimp");
const jsQR = require("jsqr");
const qrcode = require("qrcode-terminal");

const rootDir = path.resolve(__dirname, "..", "..");
const runtimeDir = path.join(rootDir, "runtime");
const outputPath = path.join(runtimeDir, "stream.env");
const qrOutputPath = path.join(runtimeDir, "douyu-login-qr.png");
const roomUrl = process.env.DOUYU_SCAN_ROOM_URL || "https://www.douyu.com/creator/main/live";
const browserPath =
  process.env.DOUYU_SCAN_BROWSER_PATH || "/usr/bin/chromium" || "/usr/bin/chromium-browser";
const headless = String(process.env.DOUYU_SCAN_HEADLESS || "true").toLowerCase() !== "false";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
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
    headless,
    executablePath: fs.existsSync(browserPath) ? browserPath : undefined,
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
    console.log("launch succeeded, opening https://passport.douyu.com/");
    const page = await browser.newPage();

    const context = browser.defaultBrowserContext();
    await context.overridePermissions("https://www.douyu.com", ["clipboard-write", "clipboard-read"]);

    await page.setRequestInterception(true);
    page.on("request", (request) => {
      if (request.resourceType() === "image") {
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
      console.log("scan this QR code:");
      qrcode.generate(qrText, { small: true });
    } else {
      console.log("qr decode failed, use the saved image instead");
    }

    console.log(`waiting for QR scan (headless=${headless})`);
    await page.waitForFunction('document.URL.includes("https://www.douyu.com/")', { timeout: 300000 });

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
      ).iterateNext();
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
      ).iterateNext();
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

    console.log(`rtmp_url: ${rtmpUrl}`);
    console.log(`stream_key: ${streamKey}`);
    await writeRuntimeEnv(rtmpUrl, streamKey);
  } finally {
    await browser.close();
  }
}

console.log("starting scan launcher");
run().catch((error) => {
  console.error(error);
  process.exit(1);
});
