import os
from collections import deque

tv_series_dir = "/home/pi/samba/电视剧/shibing/output"

# 2. 扫描并排序视频文件，将内容一次性读取到数组
current_videos_ts_list = sorted([os.path.join(root, name) for root, dirs, files in os.walk(tv_series_dir) for name in files if name.endswith(".ts")])

current_videos_ts_index = 1

while True:
    target_index = input("切换到的集数：")
    k = current_videos_ts_index - int(target_index)
    print("k:" + str(k))
    current_videos_ts_index = int(target_index)
    dq = deque(current_videos_ts_list)
    dq.rotate(k)
    current_videos_ts_list = list(dq)
    print("视频列表前4：")
    for tmp in current_videos_ts_list[:4]:
        print(tmp)