#!/bin/bash

# 检查参数是否提供正确
if [ "$#" -ne 2 ]; then
    echo "用法: $0 <目录路径> <MP4文件扩展名>"
    exit 1
fi

# 获取传递的参数
directory="$1"
mp4_extension="$2"

# 创建output文件夹
output_dir="output"

# 进入指定目录
cd "$directory"

mkdir -p "$output_dir"

# 批量转换MP4为TS文件并输出到output文件夹
for mp4_file in *."$mp4_extension"; do
    ts_file="${mp4_file%.$mp4_extension}.ts"
    echo "当前转换：$ts_file"
    ffmpeg -i "$mp4_file" -c copy -bsf:v h264_mp4toannexb "$output_dir/$ts_file"
done

echo "转换完成，文件保存在$output_dir文件夹中"
