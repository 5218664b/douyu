#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <rtmp_url> <stream_key>"
    exit 1
fi

rtmp_url="$1"
stream_key="$2"

# 拼接要写入的内容
push_line="      push $rtmp_url/$stream_key;"

# 生成临时文件
tmp_file=nginx.conf.tmp

# 复制文件前29行到临时文件
head -n 28 /home/pi/samba/douyu/conf/nginx.conf > "$tmp_file"

# 写入push行到临时文件
echo "$push_line" >> "$tmp_file"

# 复制文件第30行及其后的内容到临时文件
tail -n +30 /home/pi/samba/douyu/conf/nginx.conf >> "$tmp_file"

# 将临时文件覆盖原文件
mv "$tmp_file" /home/pi/samba/douyu/conf/nginx1.conf

sudo docker restart some-nginx

cd ../python

nohup python3 main.py  &