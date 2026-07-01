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

// Train обучает классификатор (kind, напр. "priority") на векторах+метках воркспейса.
func (c *Client) Train(ctx context.Context, wsID, kind string, vectors [][]float32, labels []int) error {
	body, _ := json.Marshal(map[string]any{"workspace_id": wsID, "kind": kind, "vectors": vectors, "labels": labels})
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

// Predict возвращает предсказанные метки + уверенность (пусто, если модели нет).
func (c *Client) Predict(ctx context.Context, wsID, kind string, vectors [][]float32) ([]int, []float32, error) {
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
		Predictions []int     `json:"predictions"`
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
