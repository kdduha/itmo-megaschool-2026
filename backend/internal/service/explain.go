package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/kdduha/itmo-megaschool-2026/backend/internal/config"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/models"
	"github.com/openai/openai-go/v3"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, value string) error
}

type ExplainService struct {
	logger       *log.Logger
	openaiClient openai.Client
	modelName    string
	cache        Cache
}

func NewExplainService(logger *log.Logger, openaiClient openai.Client, cfg config.OpenAIConfig) *ExplainService {
	return &ExplainService{
		logger:       logger,
		openaiClient: openaiClient,
		modelName:    cfg.Model,
	}
}

func (e *ExplainService) SetCacheClient(cache Cache) {
	e.cache = cache
}

func (e *ExplainService) Send(ctx context.Context, req *models.ExplainRequest) (*models.ExplainResponse, error) {
	if e.cache != nil {
		cached, found, err := e.cache.Get(ctx, getCacheKey(req))
		if err != nil {
			e.logger.Printf("cache get error: %v\n", err)
		}
		if found {
			e.logger.Println("served from cache")
			return &models.ExplainResponse{Explanation: cached}, nil
		}
	}

	params, err := e.buildOpenAIReq(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := e.openaiClient.Chat.Completions.New(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI client error: %w", err)
	}

	response := &models.ExplainResponse{
		Explanation: resp.Choices[0].Message.Content,
	}

	if e.cache != nil {
		if err := e.cache.Set(ctx, getCacheKey(req), response.Explanation); err != nil {
			e.logger.Printf("failed to set cache: %v\n", err)
		}
	}
	return response, nil
}

func (e *ExplainService) SendStream(
	ctx context.Context,
	req *models.ExplainRequest,
) (<-chan models.StreamChunk, error) {
	ch := make(chan models.StreamChunk, 1)

	if e.cache != nil {
		cached, found, err := e.cache.Get(ctx, getCacheKey(req))
		if err != nil {
			e.logger.Printf("cache get error: %v\n", err)
		}
		if found {
			ch <- models.StreamChunk{Delta: cached, Done: true}
			close(ch)
			return ch, nil
		}
	}

	params, err := e.buildOpenAIReq(req)
	if err != nil {
		return nil, fmt.Errorf("build request error: %w", err)
	}

	go func() {
		defer close(ch)

		sendOrStop := func(msg models.StreamChunk) bool {
			select {
			case ch <- msg:
				return true
			case <-ctx.Done():
				return false
			}
		}

		sendNonBlocking := func(msg models.StreamChunk) {
			select {
			case ch <- msg:
			default:
			}
		}

		stream := e.openaiClient.Chat.Completions.NewStreaming(ctx, *params)
		defer stream.Close()

		var builder strings.Builder

		for stream.Next() {
			if ctx.Err() != nil {
				sendNonBlocking(models.StreamChunk{Err: ctx.Err()})
				return
			}

			chunk := stream.Current()
			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta.Content
			if delta == "" {
				continue
			}

			builder.WriteString(delta)
			if !sendOrStop(models.StreamChunk{Delta: delta}) {
				return
			}
		}

		if err := stream.Err(); err != nil {
			sendNonBlocking(models.StreamChunk{Err: err})
			return
		}

		if e.cache != nil {
			if err := e.cache.Set(ctx, getCacheKey(req), builder.String()); err != nil {
				e.logger.Printf("failed to set cache: %v", err)
			}
		}

		sendNonBlocking(models.StreamChunk{Done: true})
	}()

	return ch, nil
}

func getCacheKey(req *models.ExplainRequest) string {
	data := []string{
		req.FileName,
		req.Prompt,
	}

	if req.Generation != nil && req.Generation.Temperature != nil {
		data = append(data, fmt.Sprintf("%f", *req.Generation.Temperature))
	}

	if req.Generation != nil && req.Generation.MaxTokens != nil {
		data = append(data, fmt.Sprintf("%d", *req.Generation.MaxTokens))
	}

	hash := sha256.Sum256([]byte(strings.Join(data, "-")))
	return hex.EncodeToString(hash[:])
}
