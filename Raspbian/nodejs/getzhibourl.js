//sudo apt install chromium
//sudo apt install Chromedriver
//sudo apt install chromium-browser
//export PUPPETEER_SKIP_DOWNLOAD=true
//npm install
const puppeteer = require('puppeteer');
const { promisify } = require('util');
const sleep = promisify(setTimeout);
const opn = require('opn');
const fs = require('fs');
const qrcode = require('qrcode-terminal');
const jsQR = require('jsqr');
var Jimp=require('jimp');


async function run() {
    const browser = await puppeteer.launch({
        //executablePath: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
        //executablePath: '/usr/bin/chromium',
        executablePath: '/usr/bin/chromium-browser',
        //headless: false,
        args: ['--start-maximized'], // 最大化窗口
    });
    try
    {
        console.log("launch成功，准备访问 https://passport.douyu.com/");
        const page = await browser.newPage();
        const version = await page.browser().version();
        console.log("version: " + version);
        // 允许剪贴板权限
        const context = browser.defaultBrowserContext();
        await context.overridePermissions("https://www.douyu.com", [
            'clipboard-write',
            'clipboard-read',
        ]);
        console.log("newPage完成");
        // 启用请求拦截
        await page.setRequestInterception(true);

        // 拦截所有请求
        page.on('request', (request) => {
            // 禁用图片和 CSS 资源的加载
            if (
                request.resourceType() === 'image' 
                // || request.resourceType() === 'stylesheet'
                ) {
            request.abort();
            } else {
            request.continue();
            }
        });

        await page.setViewport({ width: 1920, height: 1080 });
        await page.goto('https://passport.douyu.com/', { waitUntil: 'domcontentloaded' ,timeout: 0});

        console.log("获取到页面了");

        // 获取二维码
        const waitXPathSelector = '//*[@id="loginbox"]/div[1]/div[2]/div[5]/div/div[1]/div/div[1]/div/div[1]/div/canvas';
        const canvasXPath = '//*[@id="loginbox"]/div[1]/div[2]/div[5]/div/div[1]/div/div[1]/div/div[1]/div/canvas';

        await page.waitForXPath(waitXPathSelector, { timeout: 0 });
        const canvasElementHandle = await page.$x(canvasXPath);

        console.log("canvas获取成功");

        // 检查是否找到匹配的元素
        if (canvasElementHandle && canvasElementHandle.length > 0) {
            const canvas = canvasElementHandle[0];
            
            // 获取 Canvas 数据
            const canvasDataUrl = await page.evaluate(canvas => canvas.toDataURL('image/png'), canvas);

            
            // 解码 Data URL
            const [dataimage, encodedd] = canvasDataUrl.split(",", 2);
            // 在终端中显示二维码
            //qrcode.generate(encodedd, { small: true });
            const data = Buffer.from(encodedd, 'base64');

            // 保存为图片文件
            //const outputPath = "output.png";
            
            //fs.writeFileSync(outputPath, data);

            //console.log("Image saved:", outputPath);
            
            Jimp.read(data).then(function (blockimg) {
                var width=blockimg.bitmap.width,
                    height=blockimg.bitmap.height,
                    imgData=blockimg.bitmap.data;
                var code=jsQR(imgData, width, height,'invertFirst');
                if (code) {
                    console.log(code.data);//内容
                    qrcode.generate(code.data, { small: true });
            
                } else {
                    console.log('未识别成功')
                }
                }).catch(function (err2) {
                if (err2) {
                    console.log(err2);
                }
            });

            // 打开图片
            // opn(outputPath);
        } else {
            console.error('Canvas element not found.');
        }

        expected_keyword = "https://www.douyu.com/";
        await page.waitForFunction(`document.URL.includes("${expected_keyword}")`, { timeout: 300000 });

        // 用户扫描了二维码，继续执行后续操作
        console.log("User has scanned the QR code!");

        // 获取所有Cookies
        const cookies = await page.cookies();
        //console.log("All Cookies:", cookies);

        await page.goto('https://www.douyu.com/creator/main/live');

        wait = 5000; // milliseconds

        console.log("等待 即刻取消 加载");
        try {
            await page.waitForXPath('/html/body/div[4]/div/div/div/div/div/div/div[2]', { timeout: wait });
            const buttonElement = await page.$x('/html/body/div[4]/div/div/div/div/div/div/div[2]');
            await buttonElement[0].click();
        } catch (error) {
            console.log("没找到即刻取消");
        }

        console.log("等待 下一步 加载");

        try {
            await page.waitForXPath('//*[@id="root"]/div[4]/div[2]/div[2]', { timeout: wait });
            const buttonElement1 = await page.$x('//*[@id="root"]/div[4]/div[2]/div[2]');
            await buttonElement1[0].click();

            await page.waitForXPath('//*[@id="root"]/div[4]/div[2]/div[2]', { timeout: wait });
            const buttonElement2 = await page.$x('//*[@id="root"]/div[4]/div[2]/div[2]');
            await buttonElement2[0].click();

            await page.waitForXPath('//*[@id="root"]/div[4]/div[2]/div[2]', { timeout: wait });
            const buttonElement3 = await page.$x('//*[@id="root"]/div[4]/div[2]/div[2]');
            await buttonElement3[0].click();

            await page.waitForXPath('//*[@id="root"]/div[4]/div[2]/div[2]', { timeout: wait });
            const buttonElement4 = await page.$x('//*[@id="root"]/div[4]/div[2]/div[2]');
            await buttonElement4[0].click();

        } catch (error) {
            console.log("没找到下一步");
        }
        
        await sleep(1000);

        console.log("等待 开播按钮 加载");

        try {
            // 使用XPath找到按钮元素并点击
            const buttonElement = await page.waitForXPath('//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/span[1]', { timeout: 20000 });
            await buttonElement.click();

            console.log("点击 开播按钮");
            await sleep(1000);

            // 取消
            const cancelButtonElement = await page.waitForXPath('/html/body/div[4]/div/div[2]/div/div/div/div[3]/div[2]', { timeout: 20000 });
            await cancelButtonElement.click();

            console.log("点击 取消按钮");
        } catch (error) {
            console.log("未找到开播按钮");
        }

        console.log("等待 复制按钮 加载");
        // 等待 复制按钮 加载
        try {
            // 使用XPath找到按钮元素并点击
            const buttonElement = await page.waitForXPath('//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]', { timeout: 0 });
        } catch (error) {
            console.log("未找到复制按钮");
        }

        // 使用 JavaScript 获取 Canvas 数据
        const simulateClickRtmp = `
            var xpathExpression = '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]';
            var svgElement = document.evaluate(xpathExpression, document).iterateNext().lastChild;
            function simulateClick(element) {
                var event = new MouseEvent("click", {
                bubbles: true,
                cancelable: true,
                view: window
                });
                element.dispatchEvent(event);
            }
            simulateClick(svgElement);
        `;
        //await page.setClipboardInterception(true);
        await page.evaluate(simulateClickRtmp);

        await sleep(1000);
        
        const rtmp_address = await page.evaluate(() => {
            // 将剪贴板内容复制到变量
            return navigator.clipboard.readText();
        });

        console.log("rtmp_url:", rtmp_address);
        await sleep(2000);

        // 使用 JavaScript 获取 Canvas 数据
        const simulateClickzhibo = `
            var xpathExpression1 = '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[2]';
            var svgElement1 = document.evaluate(xpathExpression1, document).iterateNext().lastChild;
            function simulateClick1(element) {
                var event = new MouseEvent("click", {
                bubbles: true,
                cancelable: true,
                view: window
                });
                element.dispatchEvent(event);
            }
            simulateClick1(svgElement1);
        `;
        //await page.setClipboardInterception(true);
        await page.evaluate(simulateClickzhibo);

        const zhiboma = await page.evaluate(() => {
            // 将剪贴板内容复制到变量
            return navigator.clipboard.readText();
        });
        // ... 其他代码

        console.log("stream_push_key:", zhiboma);

        const out_url = rtmp_address + '/' + zhiboma;
        console.log("out_url:", out_url);

        // 关闭浏览器
        await browser.close();
    } catch (error) {
        console.log(error)
        await browser.close();
    }
}

console.log("开始执行");
run();
