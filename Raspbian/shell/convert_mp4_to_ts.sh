#!/bin/bash

# 检查参数是否提供正确
if [ "$#" -ne 4 ]; then
    echo "用法: $0 <目录路径> <MP4文件扩展名> <片头长度> <片尾长度>"
    exit 1
fi

# 获取传递的参数
directory="$1"
mp4_extension="$2"
movie_start_duration="$3"
movie_end_duration="$4"

# 创建output文件夹
output_dir="output"

# 进入指定目录
cd "$directory"

mkdir -p "$output_dir"

# 批量转换MP4为TS文件并输出到output文件夹
for mp4_file in *."$mp4_extension"; do
    ts_file="${mp4_file%.$mp4_extension}.ts"
    echo "当前转换：$ts_file"

    # 获取视频总时长
    total_duration=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$mp4_file")

    # 计算要保留的视频时长
    remaining_duration=$(bc <<< "$total_duration - $movie_end_duration")

    # 使用FFmpeg去掉片头和片尾，输入片头和片尾长度即可，由于关键帧的存在，可能不太准确
    # sudo ffmpeg -i /home/pi/samba/电视剧/shibing/'[www.domp4.cc]士兵突击.E08.HD1080p.mp4' -ss 170 -to 2580  -c copy -bsf:v h264_mp4toannexb ./output/'[www.domp4.cc]士兵突击.E08.HD1080p.ts'
    ffmpeg -i "$mp4_file" -ss $movie_start_duration -to $remaining_duration  -c copy -bsf:v h264_mp4toannexb "$output_dir/$ts_file"
    # ffmpeg -i "$mp4_file" -c copy -bsf:v hevc_mp4toannexb -ss 00:02:56.000 "$output_dir/$ts_file"
done

echo "转换完成，文件保存在$output_dir文件夹中"
