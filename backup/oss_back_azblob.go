package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/2018-03-28/azblob"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var logger = log.New()

var (
	Endpoint          = "oss-cn-shanghai-internal.aliyuncs.com"
	accessKeyId       = ""        //阿里云RAM账号生成
	accessKeySecret   = ""
	baseFormat        = "2006-01-02"
	bucketName        = "bihu2001"
	timeZone          = "Asia/Shanghai"
	ossFileName       = "oss_filePath.txt"
	logFile           = "get_oss.log"
	ossTxtDir         = "/mnt/osstxtfile/"
	azBlobAccountName = "bihuprodrs01diag147"
	azBlobAccountKey  = ""     //从azure blob 上获取
)

func HandleError(err error) {
	if err != nil {
		log.Error("error:", err)
		os.Exit(-1)
	}
}

func LogError(err error) {
	if err != nil {
		log.Error("error:", err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func MKDIR(dirPath string) {
	flag, err := PathExists(dirPath)
	HandleError(err)
	if flag == true {
		log.Warn(fmt.Sprintf("目录已经存在%v", dirPath))
	} else {
		log.Info("创建目录 ", dirPath)
		err = os.Mkdir(dirPath, 0755)
		HandleError(err)
	}
}

func upLoadazBlob(fileName string) {
	credential, err := azblob.NewSharedKeyCredential(azBlobAccountName, azBlobAccountKey)
	HandleError(err)
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	containerName := "oss01"
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", azBlobAccountName, containerName))

	containerURL := azblob.NewContainerURL(*URL, p)
	blobURL := containerURL.NewBlockBlobURL(fileName)
	file, err := os.Open(fileName)
	if err != nil {
		log.Error("error:", err)
		return
	}

	log.Info("Uploading the file:", fileName)
	ctx := context.Background()

	_, err = blobURL.Upload(ctx, file, azblob.BlobHTTPHeaders{ContentType: "text/plain"}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		log.Fatal(err)
	}
	LogError(err)
}

func main() {
	var day *int
	day = flag.Int("day", 0, "-day=n ,n must be int")
	flag.Parse()
	args := os.Args
	if len(args) < 2 || args == nil {
		flag.Usage()
		return
	}

	log.SetFormatter(&log.TextFormatter{})
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	HandleError(err)
	defer file.Close()
	log.SetOutput(file)
	log.SetLevel(log.DebugLevel)

	client, err := oss.New(Endpoint, accessKeyId, accessKeySecret)
	HandleError(err)

	bucket, err := client.Bucket(bucketName)
	HandleError(err)

	loc, _ := time.LoadLocation(timeZone)

	dateString := time.Now().Format(baseFormat)
	clockString := "10-0-0"
	ossDir := "/mnt/" + dateString + "-" + clockString

	MKDIR(ossDir)
	MKDIR(ossTxtDir)

	t1 := time.Now().In(loc).Year()
	t2 := time.Now().In(loc).Month()
	t3 := time.Now().In(loc).Day()

	pastDate := time.Date(t1, t2, t3, 10, 0, 0, 0, loc)

	pre := oss.Prefix("")
	marker := oss.Marker("")

	starTime := pastDate.AddDate(0, 0, -*day)
	endTime := pastDate

	ossFileName = ossTxtDir + ossFileName + "-" + starTime.Format("2006-01-02-15-04-05") + "-" + endTime.Format("2006-01-02-15-04-05")

	fd, err := os.OpenFile(ossFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	HandleError(err)
	defer fd.Close()

	os.Chdir(ossDir)

	var fileId = 1
	//扫描整个OSS，把时间段内文件路径 存入文件中

	for {
		lor, err := bucket.ListObjects(oss.MaxKeys(1), marker, pre)
		HandleError(err)
		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)

		for _, object := range lor.Objects {
			if strings.HasSuffix(object.Key, "/") {
				log.Info("create dir: ", object.Key)
				MKDIR(object.Key)
			} else {
				//只记录上传时间为昨天到今天 之间的文件
				if object.LastModified.In(loc).After(starTime) && object.LastModified.In(loc).Before(endTime) {
					log.Info("it is file: ", object.Key)
					fd.WriteString(object.Key + "\n")
					if err != nil {
						log.Error("err: ", err.Error())
					}
				} else {
					log.Warn("第", fileId, "个文件，文件不符合备份条件 ", object.Key, "的最后更新时间是: ", object.LastModified.In(loc))
				}
			}
		}
		if !lor.IsTruncated {
			break
		}
		fileId++
	}

	//读取文件里面的oss文件，并下载

	fi, err := os.Open(ossFileName)
	if err != nil {
		log.Error("Error: %s\n", err)
		return
	}
	defer fi.Close()

	pwd, err := os.Getwd()
	HandleError(err)
	br := bufio.NewReader(fi)

	fileId = 1
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		log.Info(fmt.Sprintf("当前目录是%v,开始下载第%v个文件filename is %v ", pwd, fileId, string(a)))
		err = bucket.GetObjectToFile(string(a), string(a))
		if err != nil {
			log.Error(fmt.Sprintf("Error: %s,file is %s\n", err, string(a)))
			if len(strings.Split(string(a), "/")) > 1 {
				paths, _ := filepath.Split(string(a))
				MKDIR(paths)
				err = bucket.GetObjectToFile(string(a), string(a))
				if err != nil {
					log.Error(fmt.Sprintf("Error: %s,file is %s\n", err, string(a)))
				}
			}
		}
		fileId++
	}

	//上传文件
	fi, err = os.Open(ossFileName)
	HandleError(err)
	defer fi.Close()

	br = bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		upLoadazBlob(string(a))
	}
	log.Info("上传文件结束")
}
