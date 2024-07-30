package core

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	WeChatURL    string `yaml:"WX_URL"`
	DING_URL     string `yaml:"DING_URL"`
	DingDingSign string `yaml:"DING_SECRET"`
	MsgType      string `yaml:"MSG_TYPE"`
}

type AlertWebhook struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct { //报警结构体
	Status      string      `json:"status"` //报警状态
	Labels      Labels      `json:"labels"` //报警名称
	Annotations Annotations `json:"annotations"`
	StartsAt    string      `json:"startsAt"`
	EndsAt      string      `json:"endsAt"`
}
type Labels struct { //报警具体事项
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
	shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")
	localTime := utc.In(shanghaiLoc)
	return localTime.Format("2006年01月02日 15:04:05")
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", filename, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &config, nil
}

func sendWebhookRequest(config *Config, payload *bytes.Buffer) error {
	var req *http.Request
	var err error
	var resp *http.Response

	switch config.MsgType {
	case "wechat":
		req, err = http.NewRequestWithContext(context.Background(), "POST", config.WeChatURL, payload)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	case "dingtalk":
		timestamp := time.Now().UnixNano() / 1e6
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, config.DingDingSign)

		h := hmac.New(sha256.New, []byte(config.DingDingSign))
		h.Write([]byte(stringToSign))
		sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
		dingUrl := fmt.Sprintf("%s&timestamp=%d&sign=%s", config.DING_URL, timestamp, sign)
		req, err = http.NewRequestWithContext(context.Background(), "POST", dingUrl, payload)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("timestamp", fmt.Sprintf("%d", timestamp))
		req.Header.Set("sign", sign)
	default:
		return fmt.Errorf("unsupported message type: %s", config.MsgType)
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.URL.Path != "/webhook" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var alerts AlertWebhook
	if err := json.Unmarshal(body, &alerts); err != nil {
		http.Error(w, "Error unmarshalling JSON to struct", http.StatusBadRequest)
		return
	}
	tmpl, err := template.New("alert").Funcs(template.FuncMap{
		"BuildTime": BuildTime,
	}).Parse(wechatAlertTemplate) //
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, alerts); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}

	msg := buf.String()
	markdown := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title":   "monitoring",
			"content": msg,
			"text":    msg,
		},
	}

	message, err := json.Marshal(markdown)
	if err != nil {
		http.Error(w, "JSON marshalling error", http.StatusInternalServerError)
		return
	}
	payload := bytes.NewBuffer(message)

	config, err := LoadConfig("conf.yaml")
	if err != nil {
		http.Error(w, "Configuration loading error", http.StatusInternalServerError)
		return
	}

	if config.MsgType == "" {
		http.Error(w, "MsgType is not configured", http.StatusBadRequest)
		return
	}

	if err := sendWebhookRequest(config, payload); err != nil {
		http.Error(w, "Webhook sending error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

}
