from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

import pyperclip

import qrcode_terminal
from PIL import Image
import qrcode

import os
import base64
from pyzbar.pyzbar import decode
import time
import pyautogui
from selenium.webdriver.common.alert import Alert

# 设置chromium可执行文件和chromedriver驱动路径
options = webdriver.ChromeOptions()
options.binary_location = 'C:\Program Files\Google\Chrome\Application\chrome.exe'

#options.add_argument('--headless')
driver_path = 'D:\\gitsync\\chromedriver_win32\\chromedriver.exe'
driver = webdriver.Chrome(options=options)

# 最大化窗口便于加载全部内容
driver.maximize_window()
# 请求目标网址
driver.get('https://passport.douyu.com/')
# 睡眠3秒等待加载
# 设置等待时间为30秒
wait = WebDriverWait(driver, 30)

# 使用 JavaScript 获取 Canvas 数据
data_url_script = """
    return document.querySelector("#loginbox > div.scancode-login.js-scancode-box.status-scan > div.loginbox-scanwrap > div.scancode-middle > div > div.scancode-qrcode > div > div.loginbox-scancontent > div > div.qrcode-img > div > canvas").toDataURL('image/png');;
"""
data_url = driver.execute_script(data_url_script)

# 解码 Data URL
header, encoded = data_url.split(",", 1)
data = base64.b64decode(encoded)

# 保存为图片文件
with open("output.png", "wb") as f:
    f.write(data)

# 使用 qrcode_terminal 打印二维码到终端
# qrcode_terminal.draw("output.png")

# 打开图片
image = Image.open("output.png")

# 从图片中解码二维码
decoded_objects = decode(image)

# 打印解码结果
for obj in decoded_objects:
    qr_data = obj.data.decode("utf-8")
    print(f"QR Code Data: {qr_data}")

    #qrcode_terminal.draw(qr_data)

    # 创建 QRCode 实例
    qr = qrcode.QRCode(
        version=1,  # 控制二维码的大小，可以根据需要调整
        error_correction=qrcode.constants.ERROR_CORRECT_L,  # 错误纠正水平
        box_size=10,  # 控制二维码中每个小格的像素数
        border=4,  # 控制二维码的边框像素数
    )

    # 将数据添加到 QRCode 实例
    qr.add_data(qr_data)
    qr.make(fit=True)

    # 创建二维码图像（PIL 图像对象）
    img = qr.make_image(fill_color="black", back_color="white")

    # 保存二维码图像
    img.save("qrcode.png")

    # 显示二维码图像
    # img.show()
# # 获取图片的像素数据
# pixels = image.load()

# # 遍历像素并反转颜色
# for i in range(image.width):
#     for j in range(image.height):
#         # 获取当前像素的颜色
#         current_color = pixels[i, j]

#         # 反转颜色
#         inverted_color = (255 - current_color[0], 255 - current_color[1], 255 - current_color[2])

#         # 将反转后的颜色应用到像素
#         pixels[i, j] = inverted_color
# # 保存反转后的图片
# output_path = "inverted_output.png"
# image.save(output_path)
# # 显示图片
# image.show()


expected_keyword = "https://www.douyu.com/"
wait.until(lambda driver: expected_keyword in driver.current_url)

os.remove("output.png")

# 等待用户扫描二维码，这里以扫描后出现的某个元素为例
#qr_code_element = wait.until(EC.presence_of_element_located((By.ID, "qr-code-element")))


# 用户扫描了二维码，继续执行后续操作
print("User has scanned the QR code!")

# 获取所有Cookies
cookies = driver.get_cookies()

# 打印Cookies信息
print("All Cookies:", cookies)

driver.get('https://www.douyu.com/creator/main/live')

wait = WebDriverWait(driver, 2)

try:
    button_element = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.XPATH, '/html/body/div[4]/div/div/div/div/div/div/div[2]'))
    )
    button_element.click()
except Exception:
    print("没找到即刻取消")

time.sleep(1)

try:
    button_element = WebDriverWait(driver, 10).until(
    EC.presence_of_element_located((By.XPATH, '//*[@id="root"]/div[4]/div[2]/div[2]'))
    )
    button_element.click()

    button_element = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.XPATH, '//*[@id="root"]/div[4]/div[2]/div[2]'))
    )
    button_element.click()

    button_element = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.XPATH, '//*[@id="root"]/div[4]/div[2]/div[2]'))
    )
    button_element.click()

    button_element = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.XPATH, '//*[@id="root"]/div[4]/div[2]/div[2]'))
    )
    button_element.click()
except Exception:
    print("没找到下一步")

time.sleep(1)

try:
    # 使用XPath找到按钮元素并点击
    button_element = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.XPATH, '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/span[1]'))
    )
    button_element.click()

    # 取消
    button_element = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.XPATH, '/html/body/div[4]/div/div[2]/div/div/div/div[3]/div[2]'))
    )
    button_element.click()
except Exception:
    print("未找到开播按钮")

time.sleep(1)

# 使用 JavaScript 获取 Canvas 数据
simulateClickRtmp = """
    // 使用 XPath 查询按钮元素
    var xpathExpression = '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[1]';
    var svgElement = document.evaluate(xpathExpression,document).iterateNext().lastChild;
    // 模拟点击事件的函数
    function simulateClick(element) {
        var event = new MouseEvent("click", {
            bubbles: true,
            cancelable: true,
            view: window
        });
        element.dispatchEvent(event);
    }
    
    simulateClick(svgElement);
"""
data_url = driver.execute_script(simulateClickRtmp)
# 处理弹窗
alert = Alert(driver)
alert.accept()
print(data_url)
# 模拟按下和释放组合键 Ctrl+C
#pyautogui.hotkey('ctrl', 'c')
# 模拟按下和释放 Enter 键
#pyautogui.press('enter')

 # 将剪贴板内容复制到变量
rtmp_address = pyperclip.paste()

#time.sleep(2)

# 使用 JavaScript 获取 Canvas 数据
simulateClickzhibo = """
    // 使用 XPath 查询按钮元素
    var xpathExpression1 = '//*[@id="root"]/div[2]/div[2]/div/div[1]/div[1]/div[2]/div[3]/div[2]';
    var svgElement1 = document.evaluate(xpathExpression1,document).iterateNext().lastChild;
    // 模拟点击事件的函数
    function simulateClick1(element) {
        var event = new MouseEvent("click", {
            bubbles: true,
            cancelable: true,
            view: window
        });
        element.dispatchEvent(event);
    }
    
    simulateClick1(svgElement1);
"""
#data_url = driver.execute_script(simulateClickzhibo)

 # 将剪贴板内容复制到变量
zhiboma = pyperclip.paste()

out_url = rtmp_address + '/' + zhiboma
print("out_url: " + out_url)

time.sleep(1000)
# 关闭窗口
#driver.close()
# 退出
#driver.quit()