// Клиент embed-сайдкара (fastembed). ML-граница: POST /embed {texts,kind} → {vectors}.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	url  string
	http *http.Client
}

func New(url string) *Client {
	return &Client{url: strings.TrimRight(url, "/"), http: &http.Client{Timeout: 30 * time.Second}}
}

// Embed возвращает векторы для текстов. kind = "query" | "passage" (e5-префиксы на стороне сайдкара).
func (c *Client) Embed(ctx context.Context, texts []string, kind string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]any{"texts": texts, "kind": kind})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/embed", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed sidecar %d", resp.StatusCode)
	}
	var out struct {
		Vectors [][]float32 `json:"vectors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Vectors, nil
}

// VecLiteral — вектор → pgvector-литерал "[0.1,0.2,...]".
func VecLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// --- Фаза 3: обучаемые per-workspace классификаторы ---

// Train обучает модель (kind + task classification|regression) на векторах+метках.
func (c *Client) Train(ctx context.Context, wsID, kind, task string, vectors [][]float32, labels []float64, weights []float64) error {
	payload := map[string]any{"workspace_id": wsID, "kind": kind, "task": task, "vectors": vectors, "labels": labels}
	if weights != nil {
		payload["weights"] = weights
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/train", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("train %d", resp.StatusCode)
	}
	return nil
}

// Predict возвращает предсказания (float; классы округляем у вызывающего) + уверенность.
func (c *Client) Predict(ctx context.Context, wsID, kind string, vectors [][]float32) ([]float64, []float32, error) {
	body, _ := json.Marshal(map[string]any{"workspace_id": wsID, "kind": kind, "vectors": vectors})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/predict", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("predict %d", resp.StatusCode)
	}
	var out struct {
		Predictions []float64 `json:"predictions"`
		Confidence  []float32 `json:"confidence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, err
	}
	return out.Predictions, out.Confidence, nil
}

// ParseVecLiteral — pgvector-текст "[a,b,c]" → []float32.
func ParseVecLiteral(s string) []float32 {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]float32, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil
		}
		out = append(out, float32(f))
	}
	return out
}

// Cluster кластеризует векторы (HDBSCAN). Возвращает id кластера на каждый (-1=шум).
func (c *Client) Cluster(ctx context.Context, vectors [][]float32, minClusterSize int) ([]int, error) {
	body, _ := json.Marshal(map[string]any{"vectors": vectors, "min_cluster_size": minClusterSize})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/cluster", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cluster %d", resp.StatusCode)
	}
	var out struct {
		Labels []int `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Labels, nil
}

// --- Фаза 4 (остаток): аномалии, резюме, sentiment ---

func (c *Client) post(ctx context.Context, path string, payload any, out any) error {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sidecar %s %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Anomaly — outlier-скор 0..1 на каждый вектор (IsolationForest).
func (c *Client) Anomaly(ctx context.Context, vectors [][]float32) ([]float64, error) {
	var out struct {
		Scores []float64 `json:"scores"`
	}
	if err := c.post(ctx, "/anomaly", map[string]any{"vectors": vectors}, &out); err != nil {
		return nil, err
	}
	return out.Scores, nil
}

// Summarize — экстрактивное резюме (TextRank по предложениям).
func (c *Client) Summarize(ctx context.Context, text string, maxSentences int) (string, error) {
	var out struct {
		Summary string `json:"summary"`
	}
	if err := c.post(ctx, "/summarize", map[string]any{"text": text, "max_sentences": maxSentences}, &out); err != nil {
		return "", err
	}
	return out.Summary, nil
}

// Sentiment — тональность -1..1 на каждый текст (билингвальный лексикон).
func (c *Client) Sentiment(ctx context.Context, texts []string) ([]float64, error) {
	var out struct {
		Scores []float64 `json:"scores"`
	}
	if err := c.post(ctx, "/sentiment", map[string]any{"texts": texts}, &out); err != nil {
		return nil, err
	}
	return out.Scores, nil
}
