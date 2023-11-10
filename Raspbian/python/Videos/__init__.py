# -*- coding: utf-8 -*-
import os
# 使用collections.de现列表元素的左右循环移动
from collections import deque

class Util:
    def __init__(self, tv_series_dir):
        self.tv_series_dir = tv_series_dir

        self.current_videos_ts_index = 1
        self.current_videos_ts_list = []
        if self.current_videos_ts_list == []:
            # 2. 扫描并排序视频文件，将内容一次性读取到数组
            self.current_videos_ts_list = sorted([os.path.join(root, name) for root, dirs, files in os.walk(self.tv_series_dir) for name in files if name.endswith(".ts")])

    def get_videos_ts_one(self):
        #return self.current_videos_ts_list[self.current_videos_ts_index]
        return self.current_videos_ts_list[0]

    def insert_videos_list(self, videos_index):
        if videos_index < len(self.current_videos_ts_list):
            data_to_insert = self.current_videos_ts_list.pop(videos_index)
            self.current_videos_ts_list.insert(self.current_videos_ts_index, data_to_insert)
            print("调整下一个视频为"+data_to_insert)

    def videos_index_plus(self):
        self.current_videos_ts_index = self.current_videos_ts_index + 1
        if self.current_videos_ts_index > len(self.current_videos_ts_list):
            self.current_videos_ts_index = 1
        self.loop_move_videos_list(self.current_videos_ts_index+1)

    def get_vides_ts(self, index=0):
        return self.current_videos_ts_list[index]

    def loop_move_videos_list(self, target_index):
        k = self.current_videos_ts_index - target_index
        self.current_videos_ts_index = target_index
        dq = deque(self.current_videos_ts_list)
        dq.rotate(k)
        self.current_videos_ts_list = list(dq)
        print("视频列表前4：")
        for tmp in self.current_videos_ts_list[:4]:
            print(tmp)