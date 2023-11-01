# -*- coding: utf-8 -*-
import re
import requests
import xml.etree.ElementTree as ET

def match1(text, *patterns):
    if len(patterns) == 1:
        pattern = patterns[0]
        match = re.search(pattern, text)
        if match:
            return match.group(1)
        else:
            return None
    else:
        ret = []
        for pattern in patterns:
            match = re.search(pattern, text)
            if match:
                ret.append(match.group(1))
        return ret
def process_string(input_string):
    # 检查字符串是否以'#'开始
    if input_string.startswith('#'):
        # 去掉'#'
        input_string = input_string[1:]
    
    # 去除空格
    input_string = input_string.strip()

    return input_string

def get_stream_timestamp():
    # 设置 Nginx-RTMP 的 /stat 页面地址
    rtmp_stat_url = "http://192.168.1.80:8088/stat"

    response = requests.get(rtmp_stat_url)

    if response.status_code == 200:
        # 使用ElementTree解析XML数据
        root = ET.fromstring(response.text)

        # 查找timestamp元素
        timestamp_element = root.find(".//timestamp")

        if timestamp_element is not None:
            timestamp_value = timestamp_element.text
            #print(f"当前已推流: {timestamp_value}")
            return timestamp_value
        else:
            print("Timestamp not found in XML")
    else:
        print("Failed to retrieve RTMP status")
    return None