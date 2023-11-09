# -*- coding: utf-8 -*-
import subprocess
import os
import time
from Danmaku import DanmakuClient
import Utils
from Videos import Util
from datetime import datetime
import sys

#url = 'https://www.douyu.com/1126960'
url = 'https://www.douyu.com/9163452'

filename = 'danmu'

suffix = 'txt'

#tv_series_dir = "/home/pi/samba/douyu/tmp1"
tv_series_dir = "/home/pi/samba/电视剧/士兵突击(Soldiers Sortie)624x336.X264.AAC.350M.30集全[DVDRip]/output"

# 设置RTMP URL
rtmp_url = "rtmp://192.168.1.80:1935/live/"

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
    if os.path.exists(current_videos_ts):
        os.remove(current_videos_ts)
    os.symlink(videos_util.get_vides_ts(0), current_videos_ts)

    # nohup ffmpeg -re -stream_loop -1 -f concat -safe 0 -i "$concat_file" -c copy -bsf:a aac_adtstoasc -f flv "$rtmp_url" &
    #subprocess.Popen(["nohup", "ffmpeg", "-loglevel", "quiet", "-re", "-stream_loop", "-1", "-f", "concat", "-safe", "0", "-i", concat_file, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-f", "flv", rtmp_url])
    subprocess.Popen(["nohup", "ffmpeg", "-re", "-stream_loop", "-1", "-f", "concat", "-safe", "0", "-i", concat_file, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-f", "flv", rtmp_url])

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
            os.remove(current_videos_ts)
        os.symlink(current_video, current_videos_ts)
        result = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", current_video], stdout=subprocess.PIPE)
        duration = float(result.stdout)
        total_duration = total_duration + duration + cache_duration



if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("Usage: python update_config.py <rtmp_url> <stream_key>")
        sys.exit(1)

    rtmp_push_url = sys.argv[1]
    stream_push_key = sys.argv[2]

    with open('/home/pi/samba/douyu/conf/nginx.conf', 'r') as file:
        lines = file.readlines()

    push_line = f'      push {rtmp_push_url}/{stream_push_key};\n'
    lines.pop(28)
    lines.insert(28, push_line)

    with open('/home/pi/samba/douyu/conf/nginx.conf', 'w') as file:
        file.writelines(lines)

    subprocess.run(["sudo", "docker", "restart", "some-nginx"])
    
    main()