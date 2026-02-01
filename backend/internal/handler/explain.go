package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/models"
)

type explainService interface {
	Send(ctx context.Context, req *models.ExplainRequest) (*models.ExplainResponse, error)
	SendStream(ctx context.Context, req *models.ExplainRequest, onToken func(token string) error) error
}

type ExplainHandler struct {
	service explainService
}

func NewExplainHandler(service explainService) *ExplainHandler {
	return &ExplainHandler{
		service: service,
	}
}

// Explain godoc
// @Summary Explain diagram image
// @Description Explain architecture from image + prompt. Image is sent as base64 string in JSON.
// @Tags explain
// @Accept json
// @Produce json
// @Param request body models.ExplainRequest true "Explain request"
// @Success 200 {object} models.ExplainResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /explain [post]
func (h *ExplainHandler) Explain(w http.ResponseWriter, r *http.Request) {
	var req models.ExplainRequest
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %s", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("request validation failed: %s", err), http.StatusBadRequest)
		return
	}

	resp, err := h.service.Send(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("service error: %s", err), http.StatusInternalServerError)
		return
	}

	if err = sonic.ConfigDefault.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// ExplainStream godoc
// @Summary Stream explanation
// @Description Stream explanation tokens from image + prompt. Image is sent as base64 string in JSON.
// @Tags explain
// @Accept json
// @Produce text/event-stream
// @Param request body models.ExplainRequest true "Explain request"
// @Success 200 {object} models.StreamChunk "Stream of tokens (SSE)"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /explain/stream [post]
func (h *ExplainHandler) ExplainStream(w http.ResponseWriter, r *http.Request) {
	var req models.ExplainRequest
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %s", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("request validation failed: %s", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher := w.(http.Flusher)

	var full strings.Builder
	err := h.service.SendStream(r.Context(), &req, func(token string) error {
		full.WriteString(token)

		chunk := models.StreamChunk{Delta: token}
		data, err := sonic.Marshal(chunk)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
		flusher.Flush()
		return nil
	})
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	final := models.StreamChunk{Explanation: full.String()}
	data, err := sonic.Marshal(final)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
}
