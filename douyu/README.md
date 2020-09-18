# 说明
在树莓派的openwrt使用ffmpeg推流到斗鱼,实现自动直播电影

# 硬件
树莓派 4b (Raspberry Pi 4 Model B Rev 1.2)

固件使用 https://github.com/SuLingGG/OpenWrt-Rpi Offical (5.4)

# 设置
opkg源配置,(其他源的软件包版本不对)

src/gz openwrt_core http://127.0.0.1/snapshots/targets/bcm27xx/bcm2711/packages
src/gz openwrt_base https://mirrors.cloud.tencent.com/openwrt/snapshots/packages/aarch64_cortex-a72/base
src/gz openwrt_luci https://mirrors.cloud.tencent.com/openwrt/releases/19.07.2/packages/aarch64_cortex-a72/luci
src/gz openwrt_packages https://mirrors.cloud.tencent.com/openwrt/snapshots/packages/aarch64_cortex-a72/packages
src/gz openwrt_routing https://mirrors.cloud.tencent.com/openwrt/snapshots/packages/aarch64_cortex-a72/routing

# 使用

    opkg install node node-npm ffmpeg

    cd /root/

    git clone https://github.com/5218664b/openwrt_douyu.git

    cd openwrt_douyu

    cd douyu

    cd live

    npm init

    npm -g install forever

    npm install fluent-ffmpeg moment

    chmod +x ./push_movie_to_douyu


// 执行以下命令启动推流
// 第二个参数是视频列表，第三个是rtmp地址，第四个是推流码， 第五个是要扫描的电影所在文件夹

    bash /root/openwrt_douyu/douyu/push_movie_to_douyu 'douyuMovieList.json' 'rtmp://sendtc3.douyu.com/live/' '9163452r4ZwddgtG?wsSecret=698b4354b7a8eccf663a57af7c58ef8b&wsTime=5f637ab3&wsSeek=off&wm=0&tw=0&roirecognition=0' '/mnt/sda5/movie'

# 其他

//列出所有任务，列出来之后可以查看log文件

    forever list

//关闭指定索引的任务

    forever stop 'index'

// 测试

    ffmpeg -re -loglevel error -i /mnt/sda5/movie/恐怖游轮BD中英双字[电影天堂www.dy2018.com].mp4 -vcodec copy -acodec libmp3lame -ac 2 -ar 44100 -b:a 96k -f flv "rtmp://sendtc3.douyu.com/live/9163452r4ZwddgtG?wsSecret=698b4354b7a8eccf663a57af7c58ef8b&wsTime=5f637ab3&wsSeek=off&wm=0&tw=0&roirecognition=0"

# 总结


1.测试不支持pm2

2.自己编译ffmpeg不能--enable-libx264

3.电影仅支持mp4格式

4.推流时长在12个小时左右，会自动断开

5.ffmpeg直接copy mp4的h264流是最快的，不需要编码解码，测试在Raspbian使用x264进行解密，cpu撑不住，直接拉满，尝试使用gpu加速，没成功