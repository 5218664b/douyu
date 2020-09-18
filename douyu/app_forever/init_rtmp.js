var fs = require('fs');

function update_rtmp_url(movie_list, url, code)
{
	var deve = {
		"uid": "app",
		"append": true,
		//"watch": true,
		"script": "live.js",
		"sourceDir": "/root/openwrt_douyu/douyu/live",
		"args":[]
	};
    
    //截取字符串是因为bash shell传参，最后一个参数总是带一个'\r'
	deve.args = [movie_list, url, code.substr(0, code.length-1)];
	
	fs.writeFile('/root/openwrt_douyu/douyu/app_forever/development.json', JSON.stringify(deve), 'utf-8', function() {
		console.log(deve)
		console.log("forever 配置写入成功!");
	});
}

update_rtmp_url(process.argv[2], process.argv[3], process.argv[4]);