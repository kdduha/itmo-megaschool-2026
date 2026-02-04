package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	defaultTemperature = 0.7
	defaultMaxTokens   = 512
	backendEndpoint    = "http://localhost:8080/explain/stream"

	formatFiles = []string{"jpg", "bpmn", "drawio", "txt", "png"}
)

func main() {
	ctx := context.Background()

	var results []BenchResult
	for _, formatFile := range formatFiles {
		dataPath := filepath.Join(".", "data", formatFile)

		diagrams, _ := os.ReadDir(dataPath)

		for _, diagram := range diagrams {
			filePath := filepath.Join(dataPath, diagram.Name())
			res := benchmarkDiagram(ctx, filePath)

			if res.Err != nil {
				log.Println("ERR:", res.Err)
			} else {
				log.Printf("OK %s %v", res.File, res.Duration)
			}

			results = append(results, res)
		}
	}

	printMarkdown(results)
}

func benchmarkDiagram(ctx context.Context, filePath string) BenchResult {
	start := time.Now()

	fileRaw, err := os.ReadFile(filePath)
	if err != nil {
		return BenchResult{File: filePath, Err: err}
	}

	req := ExplainRequest{
		Prompt:     "Explain this diagram. What you see?",
		FileBase64: base64.StdEncoding.EncodeToString(fileRaw),
		FileName:   filepath.Base(filePath),
		FileFormat: filepath.Ext(filePath)[1:],
		Generation: &GenerationParams{
			Temperature: &defaultTemperature,
			MaxTokens:   &defaultMaxTokens,
		},
	}

	var full strings.Builder

	err = sendStream(ctx, req, func(c Chunk) error {
		full.WriteString(c.Delta)
		return nil
	})

	return BenchResult{
		File:     filepath.Base(filePath),
		Format:   filepath.Ext(filePath)[1:],
		Duration: time.Since(start),
		Tokens:   len(full.String()),
		Err:      err,
		Size:     int64(len(fileRaw)),
	}
}

func sendStream[T any](ctx context.Context, req T, onChunk func(Chunk) error) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal req: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, backendEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bad status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(b)),
		)
	}

	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		if payload == "{}" {
			continue
		}

		if !strings.HasPrefix(payload, "{") {
			return fmt.Errorf("unexpected payload: %s", payload)
		}

		var c Chunk
		if err := json.Unmarshal([]byte(payload), &c); err != nil {
			return err
		}

		if err := onChunk(c); err != nil {
			return err
		}
	}
}

func aggregate(results []BenchResult) map[string]Agg {
	m := map[string]Agg{}
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		a := m[r.Format]
		a.Count++
		a.TotalBytes += r.Size
		a.Total += r.Duration
		m[r.Format] = a
	}
	return m
}

func printMarkdown(results []BenchResult) {
	fmt.Println("\n## Benchmark Results\n")
	fmt.Println("| Format | Requests | Avg Time | Total Time | Avg File Size |")
	fmt.Println("|--------|----------|----------|------------|---------------|")

	agg := aggregate(results)

	var (
		totalCount    int
		totalDuration time.Duration
		totalBytes    int64
	)

	for format, a := range agg {
		avg := a.Total / time.Duration(a.Count)
		avgSize := a.TotalBytes / int64(a.Count)
		fmt.Printf("| %s | %d | %v | %v | %s |\n",
			format,
			a.Count,
			avg.Round(time.Millisecond),
			a.Total.Round(time.Millisecond),
			humanBytes(avgSize),
		)
		totalCount += a.Count
		totalDuration += a.Total
		totalBytes += a.TotalBytes
	}

	if totalCount > 0 {
		mean := totalDuration / time.Duration(totalCount)
		avgSize := totalBytes / int64(totalCount)
		fmt.Printf("| **ALL** | %d | %v | %v | %s |\n",
			totalCount,
			mean.Round(time.Millisecond),
			totalDuration.Round(time.Millisecond),
			humanBytes(avgSize),
		)
	}
}

func humanBytes(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
