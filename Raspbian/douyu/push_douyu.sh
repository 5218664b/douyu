#!/bin/bash

# 设置工作根目录
work_dir="/home/pi/samba/douyu"
video_dir="$work_dir/videos"
#tv_series_dir="/home/pi/samba/douyu/tmp1"
tv_series_dir="/home/pi/samba/电视剧/士兵突击(Soldiers Sortie)624x336.X264.AAC.350M.30集全[DVDRip]"
concat_file="$work_dir/videos.txt"
video_list_file="$work_dir/videos_list.txt"
current_videos_file="$video_dir/current_videos.txt"

# 设置RTMP URL
rtmp_url="rtmp://192.168.1.80:1935/live/"

# 设置cache文件
cache_file="$video_dir/cache.mp4"

# 设置等待时间减去的秒数
wait_duration=5

# 0. 设置工作根目录
cd "$work_dir"

# 1. 生成videos.txt文件
echo "file '$video_dir/cache.mp4'" > "$concat_file"
echo "file '$video_dir/current_videos.mp4'" >> "$concat_file"

for ((i = 0; i < 9999; i++)); do
  echo "file '$video_dir/current_videos.mp4'" >> "$concat_file"
  echo "file '$video_dir/cache.mp4'" >> "$concat_file"
done

# 2. 扫描并排序视频文件，将内容一次性读取到数组
mapfile -t video_files < <(find "$tv_series_dir" -type f | sort)

ffmpeg -re -stream_loop -1 -f concat -safe 0 -i "$concat_file" -c copy -f flv "$rtmp_url" &

cache_duration=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$cache_file")

# 3. 持续循环播放视频
while true; do
  # 4. 逐个推流视频文件
  for current_video in "${video_files[@]}"; do
    echo "$current_video" > "$current_videos_file"
    ln -sf "$current_video" "$video_dir/current_videos.mp4"
    duration=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$current_video")
    sleep_duration=$(echo "$duration + $cache_duration - $wait_duration" | bc)
    sleep $sleep_duration
  done
done
