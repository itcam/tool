package main

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/scorredoira/email"
	log "github.com/sirupsen/logrus"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	accessKey    = ""               //阿里云ACCESS_KEY
	accessSecret = "" //阿里云ACCESS_SECRET
	logFile      = "slowlog.log"
	emailHost    = "smtp.exmail.qq.com:587"
	emailUser    = ""
	emailPass    = ""
)

var logger = log.New()

func init() {
	//设置输出样式，自带的只有两种样式logrus.JSONFormatter{}和logrus.TextFormatter{}
	log.SetFormatter(&log.TextFormatter{})
	//设置output,默认为stderr,可以为任何io.Writer，比如文件*os.File
	log.SetOutput(os.Stdout)
	//设置最低loglevel
	log.SetLevel(log.InfoLevel)
}

func SetBackupTime() (star string, end string) {
	t1 := time.Now().Year()
	t2 := time.Now().Month()
	t3 := time.Now().Day()
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Info("err: ", err.Error())
	}
	startTime := time.Date(t1, t2, t3, 00, 01, 00, 00, shanghai).In(time.UTC).Format("2006-01-02T15:04Z")
	endTime := time.Date(t1, t2, t3, 23, 59, 59, 00, shanghai).In(time.UTC).Format("2006-01-02T15:04Z")
	return startTime, endTime
}

func HasEle(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetSQLSlowRecord(id string) (result []map[string]string) {
	// 创建ecsClient实例
	rdsClient, err := rds.NewClientWithAccessKey(
		"cn-shanghai", // 您的可用区ID
		accessKey,     // 您的Access Key ID
		accessSecret)  // 您的Access Key Secret
	if err != nil {
		// 异常处理
		panic(err)
	}

	// 创建API请求并设置参数
	request := rds.CreateDescribeSlowLogRecordsRequest()
	request.DBInstanceId = id

	request.StartTime, request.EndTime = SetBackupTime()

	//发起请求并处理异常
	response, err := rdsClient.DescribeSlowLogRecords(request)
	if err != nil {
		log.Error(err)
		panic(err)
	}

	var TotalMap []map[string]string
	var SqlTempList []string

	for _, v := range response.Items.SQLSlowRecord {

		if strings.HasPrefix(v.SQLText, "select") {
			map1 := map[string]string{}
			map1["SQLTXT"] = v.SQLText
			map1["DBName"] = v.DBName
			map1["QueryTimes"] = v.QueryTimes
			map1["ParseRowCounts"] = strconv.Itoa(v.ParseRowCounts)
			map1["ReturnRowCounts"] = strconv.Itoa(v.ReturnRowCounts)
			map1["ExecutionStartTime"] = v.ExecutionStartTime
			if HasEle(SqlTempList, v.SQLText) == false {
				TotalMap = append(TotalMap, map1)
				SqlTempList = append(SqlTempList, v.SQLText)
			}
		}

	}
	return TotalMap
}

func sendEmail(toUserList []string, fromName, sub, body string) {
	m := email.NewHTMLMessage(sub, body)
	m.From = mail.Address{Name: fromName, Address: "git@gittab.com"}
	m.To = toUserList
	serverName := emailHost
	host, _, _ := net.SplitHostPort(serverName)
	auth := smtp.PlainAuth("", emailUser, emailPass, host)
	log.Info(fmt.Sprintf("开始发送邮件,收件人是%s", m.To))
	if err := email.Send(serverName, auth, m); err != nil {
		log.Fatal(err)
	} else {
		log.Println("发送成功")
	}
}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Error(err)
	}
	defer file.Close()
	log.SetOutput(file)
	log.SetLevel(log.DebugLevel)
	a := GetSQLSlowRecord("rm-uf6cl1p52c4nn0jy4")
	d := ""
	for i, k := range a {
		d = d + "<tr>" +
			"<td>" + strconv.Itoa(i) + "</td>" +
			"<td>" + k["DBName"] + "</td>" +
			"<td>" + k["SQLTXT"] + "</td>" +
			"<td>" + k["QueryTimes"] + "</td>" +
			"<td>" + k["ParseRowCounts"] + "</td>" +
			"<td>" + k["ReturnRowCounts"] + "</td>" +
			"<td>" + k["ExecutionStartTime"] + "</td>" +
			"</str>"
	}

	table := `<table border="7" bordercolor="black" cellspacing="0" cellpadding="0">
	<caption>数据库慢查询,部分SQL过长可能会被截断</caption>
	<tr>
	  <td width="50"><strong>序号</strong></td>
	  <td width="100"><strong>数据库名</strong></td>
	  <td width="400"><strong>SQL</strong></td>
	  <td width="150"><strong>查询时间(s)</strong></td>
	  <td width="100"><strong>扫描行数</strong></td>
	  <td width="100"><strong>返回行数</strong></td>
	  <td width="200"><strong>开始执行时间(utc时区)</strong></td>
	</tr>` + d + `
	</table>
	`
	log.Info(table)

	fname := "s.html"
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
	if err != nil {
		log.Error(err)
	}
	f.WriteString(table)
	f.Close()
	mailToUser := []string{""}
	subject := fmt.Sprintf("%s 今日生产数据库慢查询统计", time.Now().Format("2006-01-02 15:04:05"))
	sendEmail(mailToUser, "🇨🇳", subject, table)
}
