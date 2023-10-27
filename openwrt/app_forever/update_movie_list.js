var fs = require('fs');
var path = require('path');
//读这个文件夹下面所有的mp4文件
var movie_path = process.argv[2]
movie_path = path.resolve(movie_path);

var templete = {
	"cur": "0",
	"video": []
};

function ReadFiles(filePath){
    let state = fs.statSync(filePath);
    if(state.isFile()){
        //是文件
        //console.log(filePath)
		if (path.extname(filePath) == '.mp4')
		{
			var templete_1 = {
				"name" : "",
				"value": ""
			}
			templete_1.name = filePath;
			templete_1.value = filePath;
			templete.video.push(templete_1);
			//console.log(templete)
			//console.log(filePath +'mp4')
		}
    }else if (state.isDirectory()){
        //是文件夹
        //先读取
        let files = fs.readdirSync(filePath);
        files.forEach(file=>{
            ReadFiles(path.join(filePath,file));
        });
    }
}

function update_movie_list(arg_movie_path)
{
	ReadFiles(arg_movie_path);
	fs.writeFile('/root/openwrt_douyu/douyu/live/douyuMovieList.json', JSON.stringify(templete), 'utf-8', function() {
		console.log(templete)
		console.log("电影列表更新成功!");
	});
}

update_movie_list(movie_path);
