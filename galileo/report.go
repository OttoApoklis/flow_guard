package galileo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var GalileoClient *Reporter

// 伽利略上报客户端
type Reporter struct {
	AppID  string
	Token  string
	APIURL string
	Client *http.Client
}

// 创建上报实例
func NewReporter(appID, token, apiURL string) *Reporter {
	return &Reporter{
		AppID:  appID,
		Token:  token,
		APIURL: apiURL,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func GetReporter() *Reporter {
	return GalileoClient
}

// 数据点结构
type DataPoint struct {
	Metric    string            `json:"metric"`
	Timestamp int64             `json:"timestamp"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags"`
}

// 上报数据
func (r *Reporter) Report(data []DataPoint) error {
	payload := map[string]interface{}{
		"app_id": r.AppID,
		"token":  r.Token,
		"data":   data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	req, err := http.NewRequest("POST", r.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s, response: %s", resp.Status, string(body))
	}

	return nil
}

func (r *Reporter) ReportRateLimitEvent(path string, allowed bool) {

	if r == nil {
		return
	}

	status := "allowed"
	if !allowed {
		status = "blocked"
	}

	point := DataPoint{
		Metric:    "rate_limit.decision",
		Timestamp: time.Now().Unix(),
		Value:     1, // 计数1次
		Tags: map[string]string{
			"path":   path,
			"status": status,
			"source": "go_service",
		},
	}

	// 异步上报
	go func() {
		if err := r.Report([]DataPoint{point}); err != nil {
			log.Printf("Galileo report failed: %v", err)
		}
	}()
}
