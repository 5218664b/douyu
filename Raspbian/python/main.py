# -*- coding: utf-8 -*-
import subprocess
import os
import time
from Danmaku import DanmakuClient
import Utils
from Videos import Util
from datetime import datetime
import re
import time
import sys

#url = 'https://www.douyu.com/1126960'
url = 'https://www.douyu.com/9163452'

filename = 'danmu'

suffix = 'txt'

#tv_series_dir = "/home/pi/samba/douyu/tmp1"
tv_series_dir = "/home/pi/samba/hard02/magic/电视剧/士兵突击(Soldiers Sortie)624x336.X264.AAC.350M.30集全[DVDRip]/output"
#tv_series_dir = "/home/pi/samba/电视剧/shibing/output"

# 设置RTMP URL
rtmp_url = "rtmp://127.0.0.1:1935/live/"

# nohup python3 main.py &
def main():
    videos_util = Util(tv_series_dir)

    danmaku = DanmakuClient(url, filename + "." + suffix, videos_util)
    danmaku.start()

    # 设置工作根目录
    work_dir = "/home/pi/samba/douyu"
    video_dir = os.path.join(work_dir, "videos")
    
    concat_file = os.path.join(work_dir, "videos.txt")
    video_list_file = os.path.join(work_dir, "videos_list.txt")
    current_videos_file = os.path.join(video_dir, "current_videos.txt")
    play_videos_log = os.path.join(video_dir, "play_videos_log.txt")
    current_videos_ts = os.path.join(video_dir, "current_videos.ts")

    # 设置cache文件
    cache_file = os.path.join(video_dir, "cache.ts")

    # 设置等待时间减去的秒数
    wait_duration = 5

    # 0. 设置工作根目录
    os.chdir(work_dir)

    # 1. 生成videos.txt文件
    with open(concat_file, "w") as f:
        f.write(f"file '{video_dir}/cache.ts'\n")
        f.write(f"file '{video_dir}/current_videos.ts'\n")
        f.write(f"file '{video_dir}/cache.ts'\n")

    for _ in range(9999):
        with open(concat_file, "a") as f:
            f.write(f"file '{video_dir}/current_videos.ts'\n")
            f.write(f"file '{video_dir}/cache.ts'\n")

    # 如果目标文件已存在，先删除它
    if os.path.islink(current_videos_ts):
        print("文件存在")
        os.unlink(current_videos_ts)
    os.symlink(videos_util.get_vides_ts(0), current_videos_ts)

    # ffmpeg -re -stream_loop -1 -f concat -safe 0 -i /home/pi/samba/douyu/videos.txt -c copy -bsf:a aac_adtstoasc -f flv "rtmp://192.168.1.80:1935/live/"
    # ffmpeg -re -stream_loop -1 -f concat -safe 0 -i /home/pi/samba/douyu/videos.txt -c copy -bsf:a aac_adtstoasc -f flv "rtmp://sendtc3.douyu.com/live/9163452rOAYPpUEI?wsSecret=37efa56f639e5d1e5e243067ff6d2346&wsTime=658819a6&wsSeek=off&wm=0&tw=0&roirecognition=0&record=flv&origin=tct&txHost=sendtc3.douyu.com"
    # nohup ffmpeg -re -stream_loop -1 -f concat -safe 0 -i "$concat_file" -c copy -bsf:a aac_adtstoasc -f flv "$rtmp_url" &
    subprocess.Popen(["nohup", "ffmpeg", "-loglevel", "quiet", "-re", "-stream_loop", "-1", "-f", "concat", "-safe", "0", "-i", concat_file, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-f", "flv", rtmp_url])
    # subprocess.Popen(["nohup", "ffmpeg", "-re", "-stream_loop", "-1", "-f", "concat", "-safe", "0", "-i", concat_file, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-f", "flv", rtmp_url])
    # subprocess.Popen(["ffmpeg", "-re", "-stream_loop", "-1", "-f", "concat", "-safe", "0", "-i", concat_file, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-f", "flv", rtmp_url])

    # 获取cache文件的持续时间
    result = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", cache_file], stdout=subprocess.PIPE)
    cache_duration = float(result.stdout)

    total_duration = cache_duration
    current_video = ""

    # 3. 持续循环播放视频
    while True:
        # 4. 逐个推流视频文件
        while True:
            time.sleep(5)
            current_duration = Utils.get_stream_timestamp()
            if current_duration is None:
                continue
            #print("current_duration： " + current_duration)
            #print(f"total_duration： {total_duration:.2f}")
            if int(current_duration)/1000 > total_duration - 10:
                current_video = videos_util.get_videos_ts_one()
                videos_util.videos_index_plus()

                # 打开文件以追加写入（如果文件不存在则创建）
                with open(play_videos_log, 'a') as file:
                    # 写入内容到文件
                    file.write(f"current_duration: {(int(current_duration)/1000):.2f}" + '\n')
                    file.write(f"total_duration: {total_duration:.2f}" + '\n')
                    file.write(datetime.now().strftime("%Y-%m-%d %H:%M:%S") + ": " + current_video + '\n')
                break

        with open(current_videos_file, "w") as f:
            f.write(current_video)
        print("current_video: " + current_video)

        # 如果目标文件已存在，先删除它
        if os.path.exists(current_videos_ts):
            os.unlink(current_videos_ts)
        os.symlink(current_video, current_videos_ts)
        result = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", current_video], stdout=subprocess.PIPE)
        duration = float(result.stdout)
        total_duration = total_duration + duration + cache_duration

def run_command(command):
    process = subprocess.Popen(
        command,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        shell=True,
        bufsize=1,
        universal_newlines=True,
    )

    output = ''
    # 非阻塞方式读取标准输出和标准错误
    while True:
        # 读取标准输出
        output_line = process.stdout.readline()
        output = output + output_line
        if output_line:
            sys.stdout.write(f"Node.js Output: {output_line}")
            sys.stdout.flush()

        # 读取标准错误
        # error_line = process.stderr.readline()
        # if error_line:
        #     sys.stderr.write(f"Node.js Error: {error_line}")
        #     sys.stderr.flush()

        # 检查是否进程还在运行
        return_code = process.poll()
        if return_code is not None:
            # 进程已经结束，退出循环
            break

        # 等待一段时间，以免过度占用 CPU
        time.sleep(0.1)

    # 获取进程的标准输出和标准错误
    stdout, stderr = process.communicate()
    return return_code, output, stderr

if __name__ == '__main__':
    # if len(sys.argv) != 3:
    #     print("Usage: python update_config.py <rtmp_url> <stream_key>")
    #     sys.exit(1)

    node_rtmp_url = ''
    node_stream_push_key = ''
    
    if len(sys.argv) == 3:
        node_rtmp_url = sys.argv[1]
        node_stream_push_key = sys.argv[2]
    else:
        # 执行 Node.js 脚本
        node_command = 'node ../nodejs/getzhibourl.js'
        return_code, stdout, stderr = run_command(node_command)

        input_string = ''
        # 检查返回码
        if return_code == 0:
            input_string = stdout
            #print(f'Data from Node.js: {input_string}')
        else:
            print(f'Error running Node.js script. Stderr: {stderr}')
            exit();

        # 使用正则表达式提取 RTMP URL 和流推送密钥
        rtmp_url_match = re.search(r'rtmp_url:(.*?)\n', input_string, re.DOTALL)
        stream_push_key_match = re.search(r'stream_push_key:(.*?)\n', input_string, re.DOTALL)

        # 检查匹配结果
        if rtmp_url_match and stream_push_key_match:
            node_rtmp_url = rtmp_url_match.group(1).strip()
            node_stream_push_key = stream_push_key_match.group(1).strip()

            #print("rtmp_url:" + node_rtmp_url)
            #print("stream_push_key:" + node_stream_push_key)

            if node_rtmp_url == '' or node_stream_push_key == '':
                print("获取 rtmp_url 和 stream_push_key 失败")
                exit();
        else:
            print("获取rtmp_url_match和stream_push_key_match失败")
            exit();
    
    # 你的原始配置内容
    original_nginx_config = """load_module modules/ngx_rtmp_module.so;

    user nginx;
    worker_processes 1;

    error_log /var/log/nginx/error.log warn;
    pid /var/run/nginx.pid;

    events {
      worker_connections 1024;
    }

    rtmp_auto_push on;
    rtmp_auto_push_reconnect 1s;

    rtmp {
      access_log /var/log/nginx/access.log;
      server {
        listen 1935;
        listen [::]:1935 ipv6only=on;
        application live {
          live on;
          record off;
        }
      }
    }"""
    """
    在nginx rtmp配置的application live块中添加push配置项
    
    Args:
        original_config (str): 原始的nginx配置文本
        
    Returns:
        str: 添加了push配置后的新配置文本
    """
    # 按行分割配置内容，保留换行符以便还原格式
    lines = original_nginx_config.splitlines(keepends=True)
    
    # 标记是否进入了application live块
    in_live_block = False
    # 记录缩进级别（用于保持格式统一）
    indent_level = 0
    new_lines = []
    
    for line in lines:
        # 去除首尾空白字符（用于判断），保留原始行用于输出
        stripped_line = line.strip()
        
        # 检测是否进入application live块
        if stripped_line.startswith('application live {'):
            in_live_block = True
            # 计算缩进级别（原始行开头的空格数）
            indent_level = len(line) - len(line.lstrip())
            new_lines.append(line)
            continue
        
        # 检测是否离开application live块
        if in_live_block and stripped_line == '}':
            in_live_block = False
        
        # 如果在live块内部，且当前行是record off;，则在其后插入push配置
        if in_live_block and stripped_line == 'record off;':
            new_lines.append(line)
            # 保持和块内其他配置相同的缩进
            indent = ' ' * (indent_level + 2)
            # 添加push配置行
            push_line = f'{indent}push {node_rtmp_url}/{node_stream_push_key};\n'
            new_lines.append(push_line)
        else:
            new_lines.append(line)
    
    with open('/home/pi/samba/douyu/conf/nginx.conf', 'w', encoding='utf-8') as file:
        file.write(''.join(new_lines))

    subprocess.run(["sudo", "docker", "restart", "nginx"])
    
    main()