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
	Status  string `json:"status"`
	Message string `json:"message"`
	Config  Config `json:"text"`
}

type Config struct {
	Replicas      int32  `json:"replicas"`
	CPULimit      string `json:"cpu_limit"`
	CPURequest    string `json:"cpu_request"`
	MemoryLimit   string `json:"memory_limit"`
	MemoryRequest string `json:"memory_request"`
}

func PrometheusAPI(baseURL string, promql string, category string) (string, error) {
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
	metrics := strings.ReplaceAll(string(body), `\`, "")

	response := fmt.Sprintf("%s-\nPromQL: %s Metrics: %s\n", category, promql, metrics)
	return response, nil
}

func MetricsBuilder(prometheus string, deployment string, pods []string, namespace string, resourceInfo string, events []map[string]string, ingress string) (string, error) {
	baseURL := fmt.Sprintf("%s/api/v1/query_range", prometheus)

	podNames := strings.Join(pods, "|")
	promql_deployment_replica := fmt.Sprintf("kube_deployment_spec_replicas{deployment=\"%s\", namespace=\"%s\"}", deployment, namespace)
	deployment_replicas, err := PrometheusAPI(baseURL, promql_deployment_replica, "Deployment Replicas")
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_deployment_replica)
	}

	promql_cpu_usage := fmt.Sprintf("rate(container_cpu_usage_seconds_total{pod=~\"%s\", namespace=\"%s\"}[2m])", podNames, namespace)
	cpu_usage, err := PrometheusAPI(baseURL, promql_cpu_usage, "CPU Usage")
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_cpu_usage)
	}

	promql_ram_usage := fmt.Sprintf("avg(container_memory_usage_bytes{pod=~\"%s\", namespace=\"%s\"})", podNames, namespace)
	ram_usage, err := PrometheusAPI(baseURL, promql_ram_usage, "RAM Usage")
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_ram_usage)
	}

	promql_node_available_memory := "node_memory_MemAvailable_bytes"
	node_available_memory, err := PrometheusAPI(baseURL, promql_node_available_memory, "Node Available Memory")
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_node_available_memory)
	}

	promql_ingress_requests := fmt.Sprintf("sum(rate(nginx_ingress_controller_requests{ingress=\"%s\"}[2m]))", ingress)
	ingress_requsts, err := PrometheusAPI(baseURL, promql_ingress_requests, "HTTP Request Rate")
	if err != nil {
		return "", fmt.Errorf("error querying prometheus: %v, query: %s", err, promql_ingress_requests)
	}

	events_str := "Events of the pods-\n"
	for _, event := range events {
		events_str += fmt.Sprintf("Pod Name: %s, Event Type: %s, Event Reason: %s, Event Message: %s\n", event["pod"], event["type"], event["reason"], event["message"])
	}
	resourceInfo = fmt.Sprintf("Resource requests and limits-\n%s\n", resourceInfo)
	response := fmt.Sprintf("%s%s%s%s%s%s%s", deployment_replicas, cpu_usage, ram_usage, node_available_memory, ingress_requsts, resourceInfo, events_str)
	return string(response), nil
}

func GeminiAPI(url string, prompt string) (LLMResponse, error) {
	url = fmt.Sprintf("%s/askllm", url)

	prompt = strings.ReplaceAll(prompt, "\n", "\\n")
	prompt = strings.ReplaceAll(prompt, `"`, "")

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

	var response LLMResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return LLMResponse{}, fmt.Errorf("error unmarshalling response: %v", err)
	}
	fmt.Println(response)
	return response, nil
}
