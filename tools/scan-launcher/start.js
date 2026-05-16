import fs from "node:fs/promises";
import path from "node:path";
import { chromium } from "playwright";
import Jimp from "jimp";
import jsQR from "jsqr";
import qrcode from "qrcode-terminal";

const rootDir = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const runtimeDir = path.join(rootDir, "runtime");
const outputPath = path.join(runtimeDir, "stream.env");
const qrOutputPath = path.join(runtimeDir, "douyu-login-qr.png");

const roomUrl = process.env.DOUYU_SCAN_ROOM_URL ?? "https://www.douyu.com/creator/main/live";
const browserPath = process.env.DOUYU_SCAN_BROWSER_PATH;
const headless = (process.env.DOUYU_SCAN_HEADLESS ?? "true").toLowerCase() !== "false";

async function decodeQrFromBuffer(buffer) {
  const image = await Jimp.read(buffer);
  const { data, width, height } = image.bitmap;
  const code = jsQR(new Uint8ClampedArray(data), width, height);
  return code?.data ?? null;
}

async function writeRuntimeEnv(rtmpUrl, streamKey) {
  const relayForwardUrl = `${rtmpUrl.replace(/\/+$/, "")}/${streamKey}`;
  const lines = [
    `DOUYU_STREAMER_STREAM_RTMP_URL=${rtmpUrl}`,
    `DOUYU_STREAMER_STREAM_KEY=${streamKey}`,
    `DOUYU_RELAY_FORWARD_URL=${relayForwardUrl}`
  ];

  await fs.writeFile(outputPath, `${lines.join("\n")}\n`, "utf8");
  console.log(`wrote runtime stream env: ${outputPath}`);
}

async function fetchClipboardText(page) {
  return page.evaluate(async () => navigator.clipboard.readText());
}

async function clickIfVisible(page, selector, timeout = 3000) {
  try {
    const locator = page.locator(selector).first();
    await locator.waitFor({ state: "visible", timeout });
    await locator.click();
    return true;
  } catch {
    return false;
  }
}

async function main() {
  await fs.mkdir(runtimeDir, { recursive: true });
  const browser = await chromium.launch({
    headless,
    executablePath: browserPath || undefined,
    args: headless ? [] : ["--start-maximized"]
  });

  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 960 }
    });
    await context.grantPermissions(["clipboard-read", "clipboard-write"], {
      origin: "https://www.douyu.com"
    });

    const page = await context.newPage();
    await page.route("**/*", (route) => {
      const type = route.request().resourceType();
      if (type === "image") {
        return route.abort();
      }
      return route.continue();
    });

    console.log("opening Douyu login page");
    await page.goto("https://passport.douyu.com/", { waitUntil: "domcontentloaded", timeout: 0 });

    const qrCanvas = page.locator("canvas").first();
    await qrCanvas.waitFor({ state: "visible", timeout: 0 });

    const qrDataUrl = await qrCanvas.evaluate((node) => node.toDataURL("image/png"));
    const [, encoded] = qrDataUrl.split(",", 2);
    const qrBuffer = Buffer.from(encoded, "base64");
    await fs.writeFile(qrOutputPath, qrBuffer);
    console.log(`wrote qr image: ${qrOutputPath}`);

    const qrText = await decodeQrFromBuffer(qrBuffer);
    if (qrText) {
      console.log("scan this QR code:");
      qrcode.generate(qrText, { small: true });
    } else {
      console.log("qr decode failed, use the saved image instead");
    }

    console.log(`waiting for QR scan (headless=${headless})`);
    await page.waitForFunction(
      () => window.location.href.includes("https://www.douyu.com/"),
      { timeout: 300000 }
    );

    console.log("scan complete, opening creator live page");
    await page.goto(roomUrl, { waitUntil: "domcontentloaded", timeout: 0 });

    await clickIfVisible(page, "xpath=/html/body/div[4]/div/div/div/div/div/div/div[2]", 5000);

    for (let i = 0; i < 4; i += 1) {
      const ok = await clickIfVisible(page, "xpath=//*[@id=\"root\"]/div[4]/div[2]/div[2]", 3000);
      if (!ok) {
        break;
      }
    }

    if (await clickIfVisible(page, "xpath=//*[@id=\"root\"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/span[1]", 5000)) {
      await clickIfVisible(page, "xpath=/html/body/div[4]/div/div[2]/div/div/div/div[3]/div[2]", 5000);
    }

    await page.locator("xpath=//*[@id=\"root\"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]").waitFor({ timeout: 0 });

    await page.evaluate(() => {
      const node = document.evaluate(
        '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]',
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
      ).singleNodeValue;
      node?.lastChild?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
    });

    await page.waitForTimeout(1000);
    const rtmpUrl = await fetchClipboardText(page);

    await page.evaluate(() => {
      const node = document.evaluate(
        '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[2]',
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
      ).singleNodeValue;
      node?.lastChild?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window }));
    });

    await page.waitForTimeout(1000);
    const streamKey = await fetchClipboardText(page);

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

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
