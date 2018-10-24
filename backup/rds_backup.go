package main

import (
	"bufio"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"time"
)

const (
	accessKey     = ""               //阿里云ACCESS_KEY
	accessSecret  = "" //阿里云ACCESS_SECRET
	rdsInstanceId = "rm-uf6cl1p52c4nn0jy4"     //生产环境的RDS
	backLog       = "/data/log/create_back.log"
	backDir       = "/data/backup"
)

var logger = log.New()

type Fields log.Fields

func SetBackupTime() (star string, end string) {
	t1 := time.Now().Year()
	t2 := time.Now().Month()
	t3 := time.Now().Day()
	utc, err := time.LoadLocation("UTC")
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	startTime := time.Date(t1, t2, t3, 04, 00, 00, 00, shanghai).In(utc).Format("2006-01-02T15:04Z")
	endTime := time.Date(t1, t2, t3, 07, 55, 00, 00, shanghai).In(utc).Format("2006-01-02T15:04Z")
	return startTime, endTime

}

func GetRdsDownloadUrl(id string) (url string) {
	rdsClient, err := rds.NewClientWithAccessKey(
		"cn-shanghai", // 您的可用区ID
		accessKey,     // 您的Access Key ID
		accessSecret)  // 您的Access Key Secret
	if err != nil {
		panic(err)
	}

	// 创建API请求并设置参数
	request := rds.CreateDescribeBackupsRequest()
	request.DBInstanceId = id

	request.StartTime, request.EndTime = SetBackupTime()

	//发起请求并处理异常
	response, err := rdsClient.DescribeBackups(request)
	if err != nil {
		log.Error(err)
		panic(err)
	}
	if len(response.Items.Backup) == 1 {
		return response.Items.Backup[0].BackupDownloadURL
	}
	return
}

func ExecCmd(dir string, env []string, command string) {
	cmd := exec.Command("/bin/bash", "-c", command)
	log.Info("执行命令 ", command)
	//创建获取命令输出管道
	cmd.Dir = dir
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("Error:can not obtain stdout pipe for command:%s\n", err)
		return
	}

	//执行命令
	if err := cmd.Start(); err != nil {
		log.Error("Error:The command is err,", err)
		return
	}

	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)

	for {
		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {
			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				log.Error("Error :%s\n", err)
			}
			return
		}
		log.Println("%s\n", string(output))
	}
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		log.Error("wait:", err.Error())
		return
	}

}

func wget(dir string, env []string, downLog, url, filePath string) {
	ExecCmd(dir, env, fmt.Sprintf("wget --limit-rate=10M -o %v -c '%v' -O %v", downLog, url, filePath))
}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	file, err := os.OpenFile(backLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()
	log.SetOutput(file)
	log.SetLevel(log.DebugLevel)
	url := GetRdsDownloadUrl(rdsInstanceId)
	log.Info("下载URL是：", url)

	t1 := time.Now().Year()
	t2 := time.Now().Month()
	t3 := time.Now().Day()

	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Error(err.Error())
	}
	startTime := time.Date(t1, t2, t3, 04, 00, 00, 00, shanghai).Format("2006-01-02T15:04Z")
	fmt.Println(startTime)
	startDownTime := time.Now().Format("2006-01-02-15-04")
	wget(backDir, os.Environ(), fmt.Sprintf("/data/log/rds_down_log_%v", startDownTime), url, fmt.Sprintf("rds_prod_%v.tar.gz", startTime))
}
