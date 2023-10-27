var fs = require('fs');  // 文件操作模块
var moment = require('moment'); // 时间格式化模块
var url = process.argv[3];  // 推流地址
var code = process.argv[4];  // 推流参数

function logTime() {// 记录时间
	    return moment().format("YYYY-MM-DD h:mm:ss a");
}

function writeFile(fileName, data, cur)//写入文件utf-8格式
{
	  console.log(cur);
	  fs.writeFile(fileName, data, 'utf-8', function() {
		       console.log("文件写入成功，" + 'update cur index :' + cur);
		    });  
}
console.log(url + ' ' + code);
console.log('[' + logTime() + '] argv, json list: ' +  process.argv[2]);
var fileName = process.argv[2] ? process.argv[2] : 'xiaolifeidao.json'; // 文件列表
var logo = './logo.png';  // 水印logo
function ffmpegLive() {
	    var jstr = fs.readFileSync(fileName, 'utf-8');
	    var fl = JSON.parse(jstr);
	    console.log('[' + logTime() + '] Find ' + fl.video.length + ' Video Files !');
	    var index = fl.cur;
	    var ffmpeg = require('fluent-ffmpeg');
	    var inputPath = fl.video[index].value;
	    var outputPath = url + code;
	    fl.cur = parseInt(index) + 1;
	    if (fl.cur >= fl.video.length) {
		            fl.cur = 0;
		        }
	    writeFile(fileName, JSON.stringify(fl), fl.cur);
		
	    ffmpeg.getAvailableFormats(function(err, formats) {
		  console.log('Available formats:');
		  console.dir(formats);
		});
		
	    ffmpeg(inputPath)
	        .inputOptions(['-re','-loglevel quiet'])
	        .addOptions([
						'-vcodec copy',
			            '-acodec libmp3lame',
			            '-ac 2',
			            '-ar 44100',
			            '-b:a 96k'
			        ])
	        .format('flv')
	        .output(outputPath, {
			            end: true
			        })
	        .on('start', function(commandLine) {
			            console.log('[' + logTime() + '] Vedio "' + fl.video[index].name + '" is Pushing !');
			            console.log('[' + logTime() + '] Spawned Ffmpeg with command :' + commandLine);
			        })
	        .on('error', function(err) {
			            console.log('error: ' + err.message);
			            //console.log('stdout: ' + stdout);
			            //console.log('stderr: ' + stderr);
			        })
	        .on('end', function() {
			            console.log('[' + logTime() + '] Vedio "' + fl.video[index].name + '" Pushing is Finished !');
			        })
	        .run();
}
ffmpegLive();

