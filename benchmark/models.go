package main

import "time"

type ExplainRequest struct {
	Prompt     string `json:"prompt" example:"Explain architecture"`
	FileBase64 string `json:"file_base64" validate:"required" example:"iVBORw0KGgoAAAANSUhEUgAA..."`
	FileName   string `json:"file_name"`
	FileFormat string `json:"file_format" validate:"required" example:"png"`

	Generation *GenerationParams `json:"generation"`
}

type GenerationParams struct {
	Temperature *float64 `json:"temperature" example:"0.7" default:"0.7"`
	MaxTokens   *int     `json:"max_tokens" example:"512" default:"512"`
}

type Chunk struct {
	Delta string `json:"delta"`
	Done  bool   `json:"done"`
}

type BenchResult struct {
	File     string
	Format   string
	Duration time.Duration
	Tokens   int
	Err      error
	Size     int64
}

type Agg struct {
	Count      int
	Total      time.Duration
	TotalBytes int64
}
