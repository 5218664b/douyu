    
    const puppeteer = require('puppeteer');
    const { promisify } = require('util');
    const sleep = promisify(setTimeout);
    const fs = require('fs');
    // const open = require('open');
    const opn = require('opn');

    async function run() {
        
    const browser = await puppeteer.launch({
        executablePath: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
        //headless: false,
        args: ['--start-maximized'], // 最大化窗口
    });

    const page = await browser.newPage();
    // 允许剪贴板权限
    const context = browser.defaultBrowserContext();
    await context.overridePermissions("https://www.douyu.com", [
        'clipboard-write',
        'clipboard-read',
      ]);
    await page.setViewport({ width: 1920, height: 1080 });
    await page.goto('https://passport.douyu.com/', { waitUntil: 'domcontentloaded' });

    const waitXPathSelector = '//*[@id="loginbox"]/div[1]/div[2]/div[5]/div/div[1]/div/div[1]/div/div[1]/div/canvas';
    const canvasXPath = '//*[@id="loginbox"]/div[1]/div[2]/div[5]/div/div[1]/div/div[1]/div/div[1]/div/canvas';

    await page.waitForXPath(waitXPathSelector, { timeout: 2000 });
    const canvasElementHandle = await page.$x(canvasXPath);

    // 检查是否找到匹配的元素
    if (canvasElementHandle && canvasElementHandle.length > 0) {
        const canvas = canvasElementHandle[0];
        
        // 获取 Canvas 数据
        const canvasDataUrl = await page.evaluate(canvas => canvas.toDataURL('image/png'), canvas);

        
        // 解码 Data URL
        const [dataimage, encodedd] = canvasDataUrl.split(",", 2);
        const data = Buffer.from(encodedd, 'base64');

        // 保存为图片文件
        const outputPath = "output.png";
        
        fs.writeFileSync(outputPath, data);

        console.log("Image saved:", outputPath);
        
        // 打开图片
        opn(outputPath);
    } else {
        console.error('Canvas element not found.');
    }

    // 使用 JavaScript 获取 Canvas 数据
    // const dataUrl = await page.evaluate(() => {
    //     const canvas = document.querySelector("#loginbox > div.scancode-login.js-scancode-box.status-scan > div.loginbox-scanwrap > div.scancode-middle > div > div.scancode-qrcode > div > div.loginbox-scancontent > div > div.qrcode-img > div > canvas");
    //     console.log(canvas);
    //     return canvas;
    // });

    // console.log(dataUrl);

    expected_keyword = "https://www.douyu.com/";
    await page.waitForFunction(`document.URL.includes("${expected_keyword}")`);

    // 用户扫描了二维码，继续执行后续操作
    console.log("User has scanned the QR code!");

    // 获取所有Cookies
    //const cookies = await page.cookies();
    //console.log("All Cookies:", cookies);

    await page.goto('https://www.douyu.com/creator/main/live');

    wait = 2000; // milliseconds
    try {
        await page.waitForXPath('/html/body/div[4]/div/div/div/div/div/div/div[2]', { timeout: wait });
        const buttonElement = await page.$x('/html/body/div[4]/div/div/div/div/div/div/div[2]');
        await buttonElement[0].click();
    } catch (error) {
        console.log("没找到即刻取消");
    }

    await sleep(1000);

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

    try {
        // 使用XPath找到按钮元素并点击
        const buttonElement = await page.waitForXPath('//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/span[1]', { timeout: wait });
        await buttonElement.click();

        // 取消
        const cancelButtonElement = await page.waitForXPath('/html/body/div[4]/div/div[2]/div/div/div/div[3]/div[2]', { timeout: wait });
        await cancelButtonElement.click();
    } catch (error) {
        console.log("未找到开播按钮");
    }

    await sleep(1000);

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

    const rtmp_address = await page.evaluate(() => {
        // 将剪贴板内容复制到变量
        return navigator.clipboard.readText();
    });

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

    const out_url = rtmp_address + '/' + zhiboma;
    console.log("out_url:", out_url);

    await sleep(1000);
    await browser.close();
    }

    run();
