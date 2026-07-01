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
