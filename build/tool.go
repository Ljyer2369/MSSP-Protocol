package build

import (
	"blockEmulator/params"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
)

//主要就是生成批处理文件，用于启动所有节点

var absolute_path = getABpath()

func getABpath() string { //getABpath()函数检索包含调用它的Go源文件的目录的绝对路径。
	var abPath string
	_, filename, _, ok := runtime.Caller(1)
	if ok {
		abPath = path.Dir(filename)
	}
	return abPath
}

func GenerateBatFile(nodenum, shardnum, modID int) { //该函数生成一个批处理文件（batrun_showAll.bat），用于启动所有节点
	ofile, err := os.OpenFile("batrun_showAll.bat", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Panic(err)
	}
	defer ofile.Close()
	for i := 1; i < nodenum; i++ {
		for j := 0; j < shardnum; j++ {
			str := fmt.Sprintf("start cmd /k go run main.go -n %d -N %d -s %d -S %d -m %d \n\n", i, nodenum, j, shardnum, modID)
			ofile.WriteString(str)
		}
	}
	str := fmt.Sprintf("start cmd /k go run main.go -c -N %d -S %d -m %d \n\n", nodenum, shardnum, modID)

	ofile.WriteString(str)
	for j := 0; j < shardnum; j++ {
		str := fmt.Sprintf("start cmd /k go run main.go -n 0 -N %d -s %d -S %d -m %d \n\n", nodenum, j, shardnum, modID)
		ofile.WriteString(str)
	}
}

func GenerateVBSFile() { //该函数生成一个VBS脚本文件( batrun_HideWorker.vbs)。它使用 Windows Script Host (WSH) 来运行 Go 程序，而不显示命令提示符窗口。
	ofile, err := os.OpenFile("batrun_HideWorker.vbs", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Panic(err)
	}
	defer ofile.Close()
	nodenum := params.NodesInShard
	shardnum := params.ShardNum

	ofile.WriteString("Dim shell\nSet shell = CreateObject(\"WScript.Shell\")\n")

	for i := 1; i < nodenum; i++ {
		for j := 0; j < shardnum; j++ {
			str := fmt.Sprintf("shell.Run "+"\"go run "+absolute_path+"/main.go %d %d %d %d\", 0, false \n", i, nodenum, j, shardnum)
			ofile.WriteString(str)
		}
	}
	for j := 0; j < shardnum; j++ {
		str := fmt.Sprintf("shell.Run "+"\"go run "+absolute_path+"/main.go 0 %d %d %d\", 0, false \n", nodenum, j, shardnum)
		ofile.WriteString(str)
	}
	bfile, err := os.OpenFile("batrun_showSupervisor.bat", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Panic(err)
	}
	defer bfile.Close()
	bfile.WriteString("start " + absolute_path + "/batrun_HideWorker.vbs \n")
	str := fmt.Sprintf("start cmd /k \"cd "+absolute_path+" && go run main.go 12345678 %d 0 %d \" \n", nodenum, shardnum)
	bfile.WriteString(str)
}
