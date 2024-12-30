package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type LLMResponse struct {
	Status       string
	Message      string
	ReplicaCount int32
}

func PrometheusAPI(baseURL string, deployment string, pods string, promql string) (string, error) {
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating prometheus api request: %v", err)
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
		return "", fmt.Errorf("error sending prometheus api request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading prometheus api response body: %v", err)
	}
	return string(body), nil
}
func QueryPrometheus(prometheus string, deployment string, pods []string, namespace string, resourceInfo string) (string, error) {
	baseURL := fmt.Sprintf("%s/api/v1/query_range", prometheus)
	podNames := strings.Join(pods, "|")
	promql_deployment_replica := fmt.Sprintf("kube_deployment_spec_replicas{deployment=\"%s\", namespace=\"%s\"}", deployment, namespace)
	deployment_replicas, err := PrometheusAPI(baseURL, deployment, podNames, promql_deployment_replica)
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_deployment_replica)
	}
	promql_cpu_usage := fmt.Sprintf("avg(rate(container_cpu_usage_seconds_total{pod=~\"%s\", namespace=\"%s\"}[5m]))", podNames, namespace)
	cpu_usage, err := PrometheusAPI(baseURL, deployment, podNames, promql_cpu_usage)
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_cpu_usage)
	}
	promql_ram_usage := fmt.Sprintf("avg(container_memory_usage_bytes{pod=~\"%s\", namespace=\"%s\"})", podNames, namespace)
	ram_usage, err := PrometheusAPI(baseURL, deployment, podNames, promql_ram_usage)
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_ram_usage)
	}
	response := fmt.Sprintf("Deployment replicas - promql: %s metrics: %s\nCPU usage - promql: %s metrics: %s\nRAM usage - promql: %s metrics: %s\nResource request and limits: %s", promql_deployment_replica, deployment_replicas, promql_cpu_usage, cpu_usage, promql_ram_usage, ram_usage, resourceInfo)
	return string(response), nil
}

func GeminiAPI(url string, prompt string) (LLMResponse, error) {
	url = fmt.Sprintf("%s/askllm", url)
	prompt = strings.ReplaceAll(prompt, "\n", "\\n")
	prompt = strings.ReplaceAll(prompt, "\"", "\\\"")
	payload := fmt.Sprintf(`{"metrics": "%s"}`, prompt)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("error sending request: %v", err)
	}
	if resp.StatusCode != 200 {
		return LLMResponse{}, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("error reading response body: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return LLMResponse{}, fmt.Errorf("error unmarshalling response: %v", err)
	}
	fmt.Println(response)
	return LLMResponse{
		Status:       response["status"].(string),
		Message:      response["message"].(string),
		ReplicaCount: int32(response["replica_count"].(float64)),
	}, nil
}
