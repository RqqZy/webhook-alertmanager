package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wanghuiyt/ding"
	yaml "gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	WeChatURL     string `yaml:"WX_URL"`
	DingDingToken string `yaml:"DING_TOKEN"`
	DingDingSign  string `yaml:"DING_SECRET"`
	MsgType       string `yaml:"MSG_TYPE"`
}

type AlertWebhook struct {
	Alerts []Alert `json:"alerts"`
}
type Alert struct {
	Status      string      `json:"status"`
	Labels      Lables      `json:"labels"`
	Annotations Annotations `json:"annotations"`
	StartsAt    string      `json:"startsAt"`
	EndsAt      string      `json:"endsAt"`
}
type Lables struct {
	Alertname string `json:"alertname"`
	Instance  string `json:"instance"`
	Job       string `json:"job"`
	Pod       string `json:"pod"`
	Severity  string `json:"severity"`
}
type Annotations struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
	Message     string `json:"message"`
	Value       string `json:"value"`
}

func BuildTime(alterTime string) string {
	utc, err := time.Parse(time.RFC3339, alterTime)
	if err != nil {
		return ""
	}

	// 创建上海时区对象
	shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

	// 将UTC时间转换为上海时间
	localTime := utc.In(shanghaiLoc)

	// 格式化为所需格式
	return localTime.Format("2006年01月02日 15:04:05")
}
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s失败: %v", filename, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" && r.URL.Path == "/webhook" {
		// 直接读取并转发请求体内容

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var alerts AlertWebhook
		err = json.Unmarshal(body, &alerts)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error unmarshalling JSON to struct: %v", err)
			return
		}
		var markdown strings.Builder

		for _, alert := range alerts.Alerts {
			if alert.Status == "firing" {
				markdown.WriteString(fmt.Sprintf("\n### 业务告警 ###\n"))
				markdown.WriteString(fmt.Sprintf("**告警时间**: %s\n", BuildTime(alert.StartsAt)))
			} else if alert.Status == "resolved" {
				markdown.WriteString(fmt.Sprintf("\n### 告警恢复 ###\n"))

				markdown.WriteString(fmt.Sprintf("**恢复时间**: %s\n", BuildTime(alert.EndsAt)))
			}

			markdown.WriteString(fmt.Sprintf("**告警状态**: %s\n", alert.Status))
			markdown.WriteString(fmt.Sprintf("**告警类型**: %s\n", alert.Labels.Alertname))
			markdown.WriteString(fmt.Sprintf("**告警主题**: %s\n", alert.Annotations.Summary))
			markdown.WriteString(fmt.Sprintf("**告警详情**: %s %s\n", alert.Annotations.Description, alert.Annotations.Message))
			markdown.WriteString(fmt.Sprintf("**告警级别**: %s\n", alert.Labels.Severity))
			markdown.WriteString(fmt.Sprintf("**故障主机**: %s %s\n", alert.Labels.Instance, alert.Labels.Pod))

		}
		WebhookConfig, err := LoadConfig("conf.yaml")
		if err != nil {
			panic(err)
		}
		ChatType(WebhookConfig.MsgType, WebhookConfig.WeChatURL, WebhookConfig.DingDingToken, WebhookConfig.DingDingSign, markdown, w)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func ChatType(urlType string, wxkUrl string, dingToken string, dingSecret string, msg strings.Builder, w http.ResponseWriter) {
	if urlType == "" {
		panic("urlType is nil")
	}
	var s = msg.String()
	fmt.Println(s)
	switch urlType {
	case "wechat":
		wxPayload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": msg.String(),
			},
		}
		jsonPayload, _ := json.Marshal(wxPayload)
		resp, err := http.Post(wxkUrl, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("Error posting to WeChat webhook: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		bodyResponse, _ := ioutil.ReadAll(resp.Body)
		log.Printf("WeChat webhook response: %s", bodyResponse)
	case "dingtalk":
		d := ding.Webhook{AccessToken: dingToken, Secret: dingSecret}
		err := d.SendMessageText(msg.String())
		if err != nil {
			log.Printf("Error posting to DingTalk webhook: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		log.Printf("Unsupported message type: %s", urlType)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
