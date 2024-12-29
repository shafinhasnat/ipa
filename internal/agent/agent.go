package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func PrometheusAPI(baseURL string, deployment string, pods string, promql string) (string, error) {
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Add("query", promql)

	now := time.Now().UTC()
	end := now.Format(time.RFC3339)
	start := now.Add(-5 * time.Minute).Format(time.RFC3339)

	q.Add("start", start)
	q.Add("end", end)
	q.Add("step", "60s")

	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
func QueryPrometheus(prometheus string, deployment string, pods string, namespace string) (string, error) {
	baseURL := fmt.Sprintf("%s/api/v1/query_range", prometheus)
	promql_cpu_usage := fmt.Sprintf("avg(rate(container_cpu_usage_seconds_total{pod=\"%s\"}[5m]))", pods)
	cpu_usage, err := PrometheusAPI(baseURL, deployment, pods, promql_cpu_usage)
	if err != nil {
		return "", err
	}
	promql_ram_usage := fmt.Sprintf("avg(container_memory_usage_bytes{pod=\"%s\"})", pods)
	ram_usage, err := PrometheusAPI(baseURL, deployment, pods, promql_ram_usage)
	if err != nil {
		return "", err
	}
	response := fmt.Sprintf("CPU usage - promql: %s metrics: %s\nRAM usage - promql: %s metrics: %s\n", promql_cpu_usage, cpu_usage, promql_ram_usage, ram_usage)
	return string(response), nil
}

func AskLLM(prometheusData string, apikey string) (int32, error) {
	// fmt.Println(prometheusData)
	requestlimit := "limits.cpu: 500m,requests.cpu=200m"
	prompt := fmt.Sprintf("Here are some data of cpu usage by a deployment - %s\nRequest limit is %s. I need to know how many replicas are needed to handle the load smoothly. Just give me the number. Nothing else.", prometheusData, requestlimit)
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"

	// Create request body
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return -1, err
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return -1, err
	}

	// Add headers and query params
	q := req.URL.Query()
	q.Add("key", apikey)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return -1, err
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return -1, err
	}
	replicas := response.Candidates[0].Content.Parts[0].Text
	replicas = strings.Trim(replicas, "\n")
	replica, err := strconv.Atoi(replicas)
	if err != nil {
		return -1, err
	}
	r := int32(replica)
	fmt.Println(r)
	return r, nil
}
