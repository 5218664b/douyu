# -*- coding: utf-8 -*-
import subprocess
import os
import time
from Danmaku import DanmakuClient
import Utils

#url = 'https://www.douyu.com/1126960'
url = 'https://www.douyu.com/9163452'

filename = 'danmu'

suffix = 'txt'

def main():
    danmaku = DanmakuClient(url, filename + "." + suffix)
    danmaku.start()

    # 设置工作根目录
    work_dir = "/home/pi/samba/douyu"
    video_dir = os.path.join(work_dir, "videos")
    tv_series_dir = "/home/pi/samba/douyu/tmp1"
    #tv_series_dir = "/home/pi/samba/电视剧/士兵突击(Soldiers Sortie)624x336.X264.AAC.350M.30集全[DVDRip]/output"
    concat_file = os.path.join(work_dir, "videos.txt")
    video_list_file = os.path.join(work_dir, "videos_list.txt")
    current_videos_file = os.path.join(video_dir, "current_videos.txt")
    current_videos_ts = os.path.join(video_dir, "current_videos.ts")

    # 设置RTMP URL
    rtmp_url = "rtmp://192.168.1.80:1935/live/"

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

    # 2. 扫描并排序视频文件，将内容一次性读取到数组
    video_files = sorted([os.path.join(root, name) for root, dirs, files in os.walk(tv_series_dir) for name in files if name.endswith(".ts")])

    # 如果目标文件已存在，先删除它
    if os.path.exists(current_videos_ts):
        os.remove(current_videos_ts)
    os.symlink(video_files[0], current_videos_ts)

    # nohup ffmpeg -re -stream_loop -1 -f concat -safe 0 -i "$concat_file" -c copy -bsf:a aac_adtstoasc -f flv "$rtmp_url" &
    subprocess.Popen(["nohup", "ffmpeg", "-re", "-stream_loop", "-1", "-f", "concat", "-safe", "0", "-i", concat_file, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-f", "flv", rtmp_url])

    # 获取cache文件的持续时间
    result = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", cache_file], stdout=subprocess.PIPE)
    cache_duration = float(result.stdout)

    total_duration = cache_duration
    
    # 3. 持续循环播放视频
    while True:
        # 4. 逐个推流视频文件
        for current_video in video_files:
            while True:
                time.sleep(5)
                current_duration = Utils.get_stream_timestamp()
                if current_duration is None:
                    continue
                #print("current_duration： " + current_duration)
                #print(f"total_duration： {total_duration:.2f}")
                if int(current_duration)/1000 > total_duration - 10:
                    break

            with open(current_videos_file, "w") as f:
                f.write(current_video)
            # 如果目标文件已存在，先删除它
            if os.path.exists(current_videos_ts):
                os.remove(current_videos_ts)
            os.symlink(current_video, current_videos_ts)
            result = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", current_video], stdout=subprocess.PIPE)
            duration = float(result.stdout)
            total_duration = total_duration + duration + cache_duration

if __name__ == '__main__':
    main()
