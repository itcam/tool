package main

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/scorredoira/email"
	log "github.com/sirupsen/logrus"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

var (
	timeZone             = "Asia/Shanghai"
	ecsMaxIntranetInRate = 20 * 1024 * 1024 * 8 //单位bit/s
	collectInterval      = "60"                 //秒
	FullloadThreshold    = 0.4
	CpuThreshold         = 0.7
	MemThreshold         = 0.9
	EmptyloadThreshold   = 0.1
	logFile              = "/data/log/report_ecs_rs_load.log"
)

func sendEmail(toUserList []string, fromName, sub, body string) {
	m := email.NewHTMLMessage(sub, body)
	m.From = mail.Address{Name: fromName, Address: "git@gittab.com"}
	m.To = toUserList
	emailHost := "smtp.exmail.qq.com:587"
	emailUser := ""
	emailPass := ""
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

func EnvTag(list []ecs.Tag, key string) string {
	for _, v := range list {
		if v.TagKey == key {
			return v.TagValue
		}
	}
	return "N/A"

}

type Metric struct {
	Timestamp  int     `json:"timestamp"`
	UserId     string  `json:"userId"`
	InstanceId string  `json:"instanceId"`
	Minimum    float64 `json:"Minimum"`
	Average    float64 `json:"Average"`
	Maximum    float64 `json:"Maximum"`
}

type MetricResult struct {
	InstanceName    string
	InstanceIp      string
	TotalLoad       float64
	CpuUsed         float64
	MemUsed         float64
	IntranetInRate  float64
	IntranetOutRate float64
	Env             string
}

func getCpuUsed(request *cms.QueryMetricListRequest, client *cms.Client, instanceId string) float64 {
	request.Dimensions = fmt.Sprintf("{'instanceId':'%s'}", instanceId)
	request.Metric = "CPUUtilization"
	response, err := client.QueryMetricList(request)
	if err != nil {
		log.Fatal(err)
	}

	metricData := response.Datapoints

	if len(metricData) > 0 {
		var jsonData = []byte(metricData)
		var metric []Metric
		err = json.Unmarshal(jsonData, &metric)
		if err != nil {
			panic(err)
		}

		var metricSum float64
		for _, k := range metric {
			metricSum += k.Average
		}

		metricAvg, err := strconv.ParseFloat(fmt.Sprintf("%.2f", metricSum/float64(len(metric))/100), 64)
		if err != nil {
			log.Fatal(err)
		}
		return metricAvg
	} else {
		return 0
	}

}

func getMemUsed(request *cms.QueryMetricListRequest, client *cms.Client, instanceId string) float64 {
	request.Dimensions = fmt.Sprintf("{'instanceId':'%s'}", instanceId)
	request.Metric = "memory_usedutilization"
	response, err := client.QueryMetricList(request)
	if err != nil {
		log.Fatal(err)
	}
	metricData := response.Datapoints
	if len(metricData) > 0 {
		var jsonData = []byte(metricData)
		var metric []Metric
		err = json.Unmarshal(jsonData, &metric)
		if err != nil {
			panic(err)
		}
		var metricSum float64
		for _, k := range metric {
			metricSum += k.Average
		}

		metricAvg, err := strconv.ParseFloat(fmt.Sprintf("%.2f", metricSum/float64(len(metric))/100), 64)
		if err != nil {
			log.Fatal(err)
		}
		return metricAvg
	} else {
		return 0
	}

}
func getIntranetInRate(request *cms.QueryMetricListRequest, client *cms.Client, instanceId string) float64 {
	request.Dimensions = fmt.Sprintf("{'instanceId':'%s'}", instanceId)
	request.Metric = "IntranetInRate"
	response, err := client.QueryMetricList(request)
	if err != nil {
		log.Fatal(err)
	}

	metricData := response.Datapoints

	if len(metricData) > 0 {
		var jsonData = []byte(metricData)
		var metric []Metric
		err = json.Unmarshal(jsonData, &metric)
		if err != nil {
			panic(err)
		}

		var metricSum float64
		for _, k := range metric {
			metricSum += k.Average
		}

		metricAvg, err := strconv.ParseFloat(fmt.Sprintf("%.4f", metricSum/float64(len(metric))/float64(ecsMaxIntranetInRate)), 64)
		if err != nil {
			log.Fatal(err)
		}
		return metricAvg
	} else {
		return 0
	}

}
func getIntranetOutRate(request *cms.QueryMetricListRequest, client *cms.Client, instanceId string) float64 {
	request.Dimensions = fmt.Sprintf("{'instanceId':'%s'}", instanceId)
	request.Metric = "IntranetOutRate"
	response, err := client.QueryMetricList(request)
	if err != nil {
		log.Fatal(err)
	}

	metricData := response.Datapoints

	if len(metricData) > 0 {
		var jsonData = []byte(metricData)
		var metric []Metric
		err = json.Unmarshal(jsonData, &metric)
		if err != nil {
			panic(err)
		}

		var metricSum float64
		for _, k := range metric {
			metricSum += k.Average
		}

		metricAvg, err := strconv.ParseFloat(fmt.Sprintf("%.4f", metricSum/float64(len(metric))/float64(ecsMaxIntranetInRate)), 64)
		if err != nil {
			log.Fatal(err)
		}
		return metricAvg
	} else {
		return 0
	}

}

func BubbleZsort(List []MetricResult) []MetricResult {
	for i := 0; i < len(List)-1; i++ {
		for j := i + 1; j < len(List); j++ {
			if List[i].TotalLoad < List[j].TotalLoad {
				List[i], List[j] = List[j], List[i]
			}
		}
	}
	return List
}

func BubbleAsort(List []MetricResult) []MetricResult {
	for i := 0; i < len(List)-1; i++ {
		for j := i + 1; j < len(List); j++ {
			if List[i].TotalLoad > List[j].TotalLoad {
				List[i], List[j] = List[j], List[i]
			}
		}
	}
	return List
}

func HasTagEnv(i ecs.Instance) bool {

	for _, v := range i.Tags.Tag {
		if v.TagKey == "env" {
			return true
		}
	}
	return false

}

func IsOverLoad(t, c, m, threshold_t, threshold_c, threshold_m float64) string {
	if t >= threshold_t {
		return "<font color='red' face='verdana'>是</font>"
	} else {
		if c > threshold_c {
			return "<font color='red' face='verdana'>CPU过载</font>"

		}
		if m > threshold_m {
			return "<font color='red' face='verdana'>内存过载</font>"

		}
		return "<font color='green' face='verdana'>否</font>"
	}

}

func IsEmptyLoad(m, threshold float64) string {
	if m < threshold {
		return "<font color='blue' face='verdana'>是</font>"
	}
	return "<font color='green' face='verdana'>否</font>"
}

func ConvHtml(r, s []MetricResult) string {

	var d string
	for i := 0; i < 10; i++ {
		d = d + `
	<tr>
		<td>` + r[i].Env + `</td>
		<td>` + r[i].InstanceName + `</td>
		<td>` + r[i].InstanceIp + `</td>
		<td>` + strconv.FormatFloat(r[i].CpuUsed, 'g', -1, 64) + `</td>
		<td>` + strconv.FormatFloat(r[i].MemUsed, 'g', -1, 64) + `</td>
		<td>` + strconv.FormatFloat(r[i].IntranetInRate, 'g', -1, 64) + `</td>
		<td>` + strconv.FormatFloat(r[i].IntranetOutRate, 'g', -1, 64) + `</td>
		<td bgcolor="#FF9933">` + strconv.FormatFloat(r[i].TotalLoad, 'g', -1, 64) + `</td>
		<td align="center">` + IsOverLoad(r[i].TotalLoad, r[i].CpuUsed, r[i].MemUsed, FullloadThreshold, CpuThreshold, MemThreshold) + `</td>
	</tr>`
	}

	var p string
	for i := 0; i < 10; i++ {
		p = p + `
	<tr>
		<td>` + s[i].Env + `</td>
		<td>` + s[i].InstanceName + `</td>
		<td>` + s[i].InstanceIp + `</td>
		<td>` + strconv.FormatFloat(s[i].CpuUsed, 'g', -1, 64) + `</td>
		<td>` + strconv.FormatFloat(s[i].MemUsed, 'g', -1, 64) + `</td>
		<td>` + strconv.FormatFloat(s[i].IntranetInRate, 'g', -1, 64) + `</td>
		<td>` + strconv.FormatFloat(s[i].IntranetOutRate, 'g', -1, 64) + `</td>
		<td bgcolor="#FF9933">` + strconv.FormatFloat(s[i].TotalLoad, 'g', -1, 64) + `</td>
		<td align="center">` + IsEmptyLoad(s[i].TotalLoad, EmptyloadThreshold) + `</td>
	</tr>`
	}

	collectInterval, _ := strconv.Atoi(collectInterval)
	collectIntervalMin := collectInterval / 60

	table := `
        <h1>数据说明：</h1>
		<p>以下数据采集自阿里云监控，每` + strconv.Itoa(collectIntervalMin) + `分钟采集一次其中的Average，求今天的数据平均值</p>
        <p>cpu使用率 = CpuUtilization</p>
        <p>内存使用率 = memory_usedutilization</p>
		<p>内网入口带宽使用率 = InternetInRate(网络流入带宽 bit/s）/ 网卡总带宽(20 * 1024 * 1024 * 8 bit/s）</p>	
		<p>内网出口带宽使用率 = InternetOutRate(网络流出带宽 bit/s）/ 网卡总带宽(20 * 1024 * 1024 * 8 bit/s）</p>
		
		<h3>平均负载计算公式:</h3>
		
		<p><b><font color="#6495ED">平均负载率 = cpu使用率 * (50%) + 内存使用率 * (40%) + 内网入口带宽使用百分比 * (5%) + 内网出口带宽使用百分比 * (5%) </font> </b></p>
		<p><b>若平均负载超过` + strconv.FormatFloat(FullloadThreshold, 'g', -1, 64) + `则认为过载</b></p>
		<p><b>若平均负载低于` + strconv.FormatFloat(EmptyloadThreshold, 'g', -1, 64) + `则认为空载</b></p>
		<p></p>
        <table border="2" bordercolor="black" cellspacing="0" cellpadding="0">
		<tr><th colspan="9" align="center" bgcolor="#FF6633">平均负载TOP 10(负载降序)</th></tr>
		<tr>
		<td width="80"><strong>环境类型</strong></td>
	  	<td width="200"><strong>主机名</strong></td>
	  	<td width="100"><strong>服务器IP</strong></td>
	  	<td width="100"><strong>CPU使用率【0-1】</strong></td>
	  	<td width="100"><strong>内存使用率【0-1】</strong></td>
		<td width="100"><strong>内网入口带宽使用率【0-1】</strong></td>
		<td width="100"><strong>内网出口带宽使用率【0-1】</strong></td>
		<td width="100" bgcolor="#FF9933"><strong>平均负载率【0-1】</strong></td>
		<td width="80" align="center" ><strong>是否过载</strong></td>
		</tr>
		` + d + `
		</table>
		<p></p>
  		<table border="2" bordercolor="black" cellspacing="0" cellpadding="0">
		<tr><th colspan="9" align="center" bgcolor="#FF6633">平均负载TOP 10(负载升序)</th></tr>
		<tr>
		<td width="80"><strong>环境类型</strong></td>
	  	<td width="200"><strong>主机名</strong></td>
	  	<td width="100"><strong>服务器IP</strong></td>
	  	<td width="100"><strong>CPU使用率【0-1】</strong></td>
	  	<td width="100"><strong>内存使用率【0-1】</strong></td>
		<td width="100"><strong>内网入口带宽使用率【0-1】</strong></td>
		<td width="100"><strong>内网出口带宽使用率【0-1】</strong></td>
		<td width="100" bgcolor="#FF9933"><strong>平均负载率【0-1】</strong></td>
		<td width="80" align="center" ><strong>是否空载</strong></td>
		</tr>
		` + p + `
		</table>
		`

	return table

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
	// 创建ecsClient实例

	ecsClient, err := ecs.NewClientWithAccessKey(
		"cn-shanghai",                    // 您的可用区ID
		"",               // 您的Access Key ID
		"") // 您的Access Key Secret
	if err != nil {
		log.Error(err)
		panic(err)
	}

	metricClient, err := cms.NewClientWithAccessKey(
		"cn-shanghai",                    // 您的可用区ID
		"",               // 您的Access Key ID
		"", // 您的Access Key Secret
	)
	if err != nil {
		log.Error(err)
		panic(err)
	}

	// 创建API请求并设置参数
	ecsInstanceRequest := ecs.CreateDescribeInstancesRequest()
	ecsInstanceRequest.PageSize = "100"
	// 发起请求并处理异常

	response, err := ecsClient.DescribeInstances(ecsInstanceRequest)
	if err != nil {
		log.Error(err)
		panic(err)
	}

	loc, _ := time.LoadLocation(timeZone)
	//nTime := time.Now().In(loc)
	//yesTime := nTime.AddDate(0, 0, -7)
	//
	startTime := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 1, 0, 0, loc).AddDate(0, 0, 0)
	endTime := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 23, 59, 0, 0, loc).AddDate(0, 0, 0)

	//获取开始时间和结束时间的毫秒时间戳
	//nTimeStamp := nTime.UnixNano() / 1e6
	//
	//yesTimeStamp := yesTime.UnixNano() / 1e6

	//获取开始时间和结束时间的毫秒时间戳
	nTimeStamp := endTime.UnixNano() / 1e6

	yesTimeStamp := startTime.UnixNano() / 1e6

	fmt.Println(nTimeStamp, yesTimeStamp)

	ecsMetricRequest := cms.CreateQueryMetricListRequest()
	ecsMetricRequest.Project = "acs_ecs_dashboard"
	ecsMetricRequest.AcceptFormat = "json"
	ecsMetricRequest.StartTime = strconv.FormatInt(yesTimeStamp, 10)
	ecsMetricRequest.EndTime = strconv.FormatInt(nTimeStamp, 10)
	ecsMetricRequest.Period = collectInterval
	ecsMetricRequest.Method = "GET"
	ecsMetricRequest.GetQueries()
	var metric MetricResult
	var result []MetricResult

	for _, i := range response.Instances.Instance {
		if HasTagEnv(i) {
			CpuUsedData := getCpuUsed(ecsMetricRequest, metricClient, i.InstanceId)
			MemUsedData := getMemUsed(ecsMetricRequest, metricClient, i.InstanceId)
			IntranetInRate := getIntranetInRate(ecsMetricRequest, metricClient, i.InstanceId)
			IntranetOutRate := getIntranetOutRate(ecsMetricRequest, metricClient, i.InstanceId)
			TotalLoad, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", CpuUsedData/10*5+MemUsedData/10*4+IntranetInRate/20+IntranetOutRate/20), 64)
			metric.InstanceName = i.InstanceName
			metric.InstanceIp = i.VpcAttributes.PrivateIpAddress.IpAddress[0]
			metric.TotalLoad = TotalLoad
			metric.CpuUsed = CpuUsedData
			metric.MemUsed = MemUsedData
			metric.IntranetInRate = IntranetInRate
			metric.IntranetOutRate = IntranetOutRate
			metric.Env = EnvTag(i.Tags.Tag, "env")
			result = append(result, metric)
			log.Info(metric)
			time.Sleep(1 * time.Second)
		}
	}
	r1 := make([]MetricResult, len(result))
	r2 := make([]MetricResult, len(result))
	copy(r1, result)
	copy(r2, result)
	r1 = BubbleZsort(r1)
	r2 = BubbleAsort(r2)
	context := ConvHtml(r1, r2)


	mailToUser := []string{"test@qq.com"}
	subject := fmt.Sprintf("%s 阿里云服务器资源使用情况", time.Now().Format("2006-01-02 15:04:05"))

	sendEmail(mailToUser, "🇨🇳", subject, context)
}
