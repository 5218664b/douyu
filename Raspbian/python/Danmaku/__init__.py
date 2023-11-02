# -*- coding: utf-8 -*-
# 部分弹幕功能代码来自项目：https://github.com/IsoaSFlus/danmaku，感谢大佬
# 快手弹幕代码来源及思路：https://github.com/py-wuhao/ks_barrage，感谢大佬
# 部分斗鱼录播修复代码与思路来源于：https://github.com/SmallPeaches/DanmakuRender，感谢大佬
# 仅抓取用户弹幕，不包括入场提醒、礼物赠送等。

import asyncio
import os
import platform
import re
import ssl
import sys
import threading
import time
import logging
from typing import Optional, List

import lxml.etree as etree
import aiohttp

from Danmaku.douyu import Douyu
#from biliup.plugins.Danmaku.huya import Huya
#from biliup.plugins.Danmaku.bilibili import Bilibili
#from biliup.plugins.Danmaku.twitch import Twitch
#from biliup.plugins.Danmaku.douyin import Douyin

from Utils import process_string
from Videos import Util

logger = logging.getLogger('biliup')


class DanmakuClient:
    class WebsocketErrorException(Exception):
        pass

    def __init__(self, url, filename, Util):
        self.__starttime = time.time()
        self.__filename = os.path.splitext(filename)[0] + '.xml'
        self.__filename_video_suffix = filename
        self.__url = ''
        self.__site = None
        self.__hs = None
        self.__ws = None
        self.__dm_queue = None
        self.__record_task: Optional[asyncio.Task] = None
        self.Util = Util

        if 'http://' == url[:7] or 'https://' == url[:8]:
            self.__url = url
        else:
            self.__url = 'http://' + url
        for u, s in {'douyu.com': Douyu,
                     #'huya.com': Huya,
                     #'live.bilibili.com': Bilibili,
                     #'twitch.tv': Twitch,
                     #'douyin.com': Douyin
                     }.items():
            if re.match(r'^(?:http[s]?://)?.*?%s/(.+?)$' % u, url):
                self.__site = s
                self.__u = u
                break

        if self.__site is None:
            # 抛出异常由外部处理 exit()会导致进程退出
            raise Exception(f"{DanmakuClient.__name__}:{self.__url}: 不支持录制弹幕")

    async def __init_ws(self):
        try:
            ws_url, reg_datas = await self.__site.get_ws_info(self.__url)
            ctx = ssl.create_default_context()
            ctx.set_ciphers('DEFAULT')
            self.__ws = await self.__hs.ws_connect(ws_url, ssl_context=ctx, headers=getattr(self.__site, 'headers', {}))
            for reg_data in reg_datas:
                if type(reg_data) == str:
                    await self.__ws.send_str(reg_data)
                else:
                    await self.__ws.send_bytes(reg_data)
        except asyncio.CancelledError:
            raise
        except Exception:
            raise self.WebsocketErrorException()

    async def __heartbeats(self):
        while self.__site.heartbeat:
            # 每隔这么长时间发送一次心跳包
            await asyncio.sleep(self.__site.heartbeatInterval)
            # 发送心跳包
            if type(self.__site.heartbeat) == str:
                await self.__ws.send_str(self.__site.heartbeat)
            else:
                await self.__ws.send_bytes(self.__site.heartbeat)

    async def __fetch_danmaku(self):
        while True:
            # 使用 async for msg in self.__ws
            # 会导致在连接断开时 需要很长时间15min或者更多才能检测到
            msg = await self.__ws.receive()

            if msg.type in [aiohttp.WSMsgType.CLOSED, aiohttp.WSMsgType.ERROR]:
                # 连接关闭的异常 会到外层统一处理
                raise self.WebsocketErrorException()

            try:
                result = self.__site.decode_msg(msg.data)

                if isinstance(result, tuple):
                    ms, ack = result
                    if ack is not None:
                        # 发送ack包
                        if type(ack) == str:
                            await self.__ws.send_str(ack)
                        else:
                            await self.__ws.send_bytes(ack)
                else:
                    ms = result

                for m in ms:
                    await self.__dm_queue.put(m)
            except asyncio.CancelledError:
                raise
            except Exception:
                logger.exception(f"{DanmakuClient.__name__}:{self.__url}: 弹幕接收异常")
                continue
                # await asyncio.sleep(10) 无需等待 直接获取下一条websocket消息
                # 这里出现异常只会是 decode_msg 的问题

    async def __print_danmaku(self):
        parser = etree.XMLParser(recover=True)
        root = etree.Element("root")
        tree = etree.ElementTree(root, parser=parser)
        msg_i = 0
        msg_col = {'0': '16777215', '1': '16717077', '2': '2000880', '3': '8046667', '4': '16744192',
                   '5': '10172916',
                   '6': '16738740'}

        def write_file(filename):
            try:
                with open(filename, "wb") as f:
                    #etree.indent(root, "\t")
                    tree.write(f, encoding="UTF-8", xml_declaration=True, pretty_print=True)
            except Exception as e:
                logger.warning(f"{DanmakuClient.__name__}:{self.__url}: 弹幕写入异常 - {e}")

        try:
            while True:
                m = await self.__dm_queue.get()
                if m.get('msg_type') == 'danmaku':
                    try:
                        d = etree.SubElement(root, 'd')
                        if 'col' in m:
                            color = msg_col[m["col"]]
                        elif 'color' in m:
                            color = m["color"]
                        else:
                            color = '16777215'
                        msg_time = format(time.time() - self.__starttime, '.3f')
                        d.set('p', f"{msg_time},1,25,{color},0,0,0,0")
                        d.text = m["content"]
                    except Exception as Error:
                        logger.warning(f"{DanmakuClient.__name__}:{self.__url}:弹幕处理异常 - {Error}")
                        # 异常后略过本次弹幕
                        continue

                    if msg_i >= 5:
                        # 收到五条弹幕后写入 减少io
                        # 可能会写入失败 会在下次五条或者任务被取消时重新尝试写入
                        write_file(self.__filename)
                        msg_i = 0
                    else:
                        msg_i = msg_i + 1
                        logger.warning(d.text)
                        if d.text.startswith('#'):
                            #self.Util.insert_videos_list(int(process_string(d.text)))
                            self.Util.loop_move_videos_list(int(process_string(d.text)))
        except asyncio.CancelledError:
            # 在这段代码中只有执行m = await self.__dm_queue.get()的时候可能会被取消 所以无需担心其他问题
            # 任务被取消之前需要写入一下 不然会丢失没写入的
            write_file(self.__filename)
            # 如果不存在对应的视频则删除弹幕
            if not (os.path.exists(f"{self.__filename_video_suffix}.part") or
                    os.path.exists(f"{self.__filename_video_suffix}")):
                os.remove(self.__filename)
            raise

    def start(self):
        init_event = threading.Event()

        async def __init():
            logger.info(f'开始弹幕录制: {self.__filename}')
            self.__record_task = asyncio.create_task(self.__run())
            init_event.set()
            try:
                await self.__record_task
            except asyncio.CancelledError:
                pass
            finally:
                # fix event loop is close
                if sys.version_info < (3, 11) and platform.system() == 'Windows':
                    await asyncio.sleep(2)
            logger.info(f'结束弹幕录制: {self.__filename}')

        threading.Thread(target=asyncio.run, args=(__init(),)).start()
        # 等待初始化完成避免未初始化完成的时候就停止任务
        init_event.wait()

    async def __run(self):
        while True:
            is_retry = False
            danmaku_tasks: Optional[List[asyncio.Task]] = None
            try:
                self.__dm_queue = asyncio.Queue()
                self.__hs = aiohttp.ClientSession()
                await self.__init_ws()
                
                danmaku_tasks = [asyncio.create_task(self.__heartbeats()),
                                 asyncio.create_task(self.__fetch_danmaku()),
                                 asyncio.create_task(self.__print_danmaku())]
                await asyncio.gather(*danmaku_tasks)
            except asyncio.CancelledError:
                raise
            except self.WebsocketErrorException:
                # 连接断开等30秒重连
                # 在关闭之前一直重试
                logger.warning(f"{DanmakuClient.__name__}:{self.__filename}: 弹幕连接异常,将在 30 秒后重试")
                is_retry = False
            except Exception as Error:
                # 记录异常不到外部处理
                logger.exception(f"{DanmakuClient.__name__}:{self.__filename}: 弹幕异常,将在 30 秒后重试")
                is_retry = True
            finally:
                if type(danmaku_tasks) is list:
                    for task in danmaku_tasks:
                        task.cancel()
                if self.__ws is not None and not self.__ws.closed:
                    await self.__ws.close()
                if self.__hs is not None and not self.__hs.closed:
                    await self.__hs.close()
                if is_retry:
                    await asyncio.sleep(30)
                    #continue
            break

    def stop(self):
        if self.__record_task is not None:
            self.__record_task.cancel()

# 虎牙直播：https://www.huya.com/lpl
# 斗鱼直播：https://www.douyu.com/9999
