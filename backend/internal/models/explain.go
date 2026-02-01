package models

import "fmt"

// ExplainRequest represents request for explain endpoint
type ExplainRequest struct {
	Prompt     string `json:"prompt" example:"Explain architecture"`
	FileBase64 string `json:"file_base64" validate:"required" example:"iVBORw0KGgoAAAANSUhEUgAA..."`
	FileName   string `json:"file_name" validate:"required" example:"diagram.png"`
	FileFormat string `json:"file_format" validate:"required" example:"png"`

	// Optional generation parameters
	Generation *GenerationParams `json:"generation"`
}

func (r ExplainRequest) Validate() error {
	if r.FileBase64 == "" {
		return fmt.Errorf("file_base64 is empty")
	}
	if r.FileName == "" {
		return fmt.Errorf("file_name is empty")
	}
	if r.FileFormat == "" {
		return fmt.Errorf("file_format is empty")
	}
	return nil
}

// GenerationParams holds optional OpenAI-like generation parameters
type GenerationParams struct {
	Temperature *float64 `json:"temperature" example:"0.7" default:"0.7"`
	MaxTokens   *int     `json:"max_tokens" example:"512" default:"512"`
}

type ExplainResponse struct {
	Explanation string `json:"explanation"`
}

type StreamChunk struct {
	Delta       string `json:"delta,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}
