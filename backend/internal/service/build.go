package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kdduha/itmo-megaschool-2026/backend/internal/metrics"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/models"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"image/jpeg"

	fitz "github.com/gen2brain/go-fitz"
)

const dpi = 120

func getUserPrompt(req *models.ExplainRequest) string {
	userPrompt := fmt.Sprintf(userPromptTemplate, req.FileName)
	if req.Prompt != "" {
		userPrompt = fmt.Sprintf("%s\nDetails and questions: %s", userPrompt, req.Prompt)
	}
	return userPrompt
}

func (e *ExplainService) buildOpenAIReq(req *models.ExplainRequest) (*openai.ChatCompletionNewParams, error) {
	var (
		duration         time.Duration
		preprocessStatus string

		messages []openai.ChatCompletionMessageParamUnion
		err      error
	)

	e.logger.Printf("start preprocessing file: %s\n", req.FileName)
	defer func() {
		e.logger.Printf("finish preprocessing file: %s\n", req.FileName)
		metrics.FilePreprocessTotal(preprocessStatus, req.FileFormat)
		metrics.FilePreprocessDuration(preprocessStatus, req.FileName, duration)
	}()

	start := time.Now()

	switch req.FileFormat {
	case PNG, JPEG, JPG:
		messages = e.buildImageMessages(req)
	case DRAWIO, BPMN, SVG:
		messages, err = e.buildDiagramMessages(req)
		if err != nil {
			preprocessStatus = "failed"
			duration = time.Duration(start.Second())
			return nil, fmt.Errorf("failed to convert diagram: %v", err)
		}
	case TXT:
		messages, err = e.buildTxtMessages(req)
		if err != nil {
			preprocessStatus = "failed"
			duration = time.Duration(start.Second())
			return nil, fmt.Errorf("failed to convert txt: %v", err)
		}
	case PDF:
		messages, err = e.buildPdfMessages(req)
		if err != nil {
			preprocessStatus = "failed"
			duration = time.Duration(start.Second())
			return nil, fmt.Errorf("failed to convert pdf: %v", err)
		}
	default:
		preprocessStatus = "failed"
		duration = time.Duration(start.Second())
		return nil, fmt.Errorf("unsupported fileformat {%s}", req.FileFormat)
	}

	params := &openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(e.modelName),
		Messages: messages,
	}

	if req.Generation != nil && req.Generation.MaxTokens != nil {
		params.MaxCompletionTokens = openai.Int(int64(*req.Generation.MaxTokens))
	}

	if req.Generation != nil && req.Generation.MaxTokens != nil {
		params.Temperature = openai.Float(*req.Generation.Temperature)
	}

	preprocessStatus = "success"
	duration = time.Duration(start.Second())
	return params, nil
}

func (e *ExplainService) buildImageMessages(req *models.ExplainRequest) []openai.ChatCompletionMessageParamUnion {
	userPrompt := getUserPrompt(req)
	imageData := fmt.Sprintf("data:image/%s;base64,%s", strings.TrimPrefix(req.FileFormat, "."), req.FileBase64)

	return []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPromptImage),
		openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(userPrompt),
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: imageData,
			}),
		}),
	}
}

func (e *ExplainService) buildDiagramMessages(req *models.ExplainRequest) ([]openai.ChatCompletionMessageParamUnion, error) {
	userPrompt := getUserPrompt(req)
	base64Img, err := convertDiagramToImageTemp(req.FileBase64, req.FileFormat)
	if err != nil {
		return nil, err
	}

	imageData := fmt.Sprintf("data:image/%s;base64,%s", JPG, base64Img)
	return []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPromptImage),
		openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(userPrompt),
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: imageData,
			}),
		}),
	}, nil
}

func (e *ExplainService) buildTxtMessages(req *models.ExplainRequest) ([]openai.ChatCompletionMessageParamUnion, error) {
	userPrompt := getUserPrompt(req)
	inputData, err := base64.StdEncoding.DecodeString(req.FileBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	userPrompt = fmt.Sprintf("%s\nDiagram text:\n%s", userPrompt, inputData)
	return []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPromptImage),
		openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(userPrompt),
		}),
	}, nil
}

func (e *ExplainService) buildPdfMessages(req *models.ExplainRequest) ([]openai.ChatCompletionMessageParamUnion, error) {
	userPrompt := getUserPrompt(req)
	inputData, err := base64.StdEncoding.DecodeString(req.FileBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	doc, err := fitz.NewFromMemory(inputData)
	if err != nil {
		return nil, fmt.Errorf("mupdf open failed: %w", err)
	}
	defer doc.Close()

	parts := []openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(userPrompt),
	}

	for n := 0; n < doc.NumPage(); n++ {
		img, err := doc.ImageDPI(n, dpi)
		if err != nil {
			return nil, fmt.Errorf("render page %d failed: %w", n, err)
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		if err != nil {
			return nil, fmt.Errorf("jpeg encode page %d failed: %w", n, err)
		}

		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
		imageData := "data:image/jpeg;base64," + encoded

		parts = append(parts,
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: imageData,
			}),
		)
	}

	return []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPromptImage),
		openai.UserMessage(parts),
	}, nil
}

func convertDiagramToImageTemp(inputBase64, fileExt string) (string, error) {
	var (
		cmdArgs []string
	)

	inputData, err := base64.StdEncoding.DecodeString(inputBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	tmpIn, err := os.CreateTemp("", fmt.Sprintf("input-*.%s", fileExt))
	if err != nil {
		return "", fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(tmpIn.Name())
	if _, err := tmpIn.Write(inputData); err != nil {
		tmpIn.Close()
		return "", fmt.Errorf("failed to write to temp input file: %w", err)
	}
	tmpIn.Close()

	outExt := PNG
	if fileExt == DRAWIO {
		outExt = JPG
	}
	tmpOut, err := os.CreateTemp("", fmt.Sprintf("output-*.%s", outExt))
	if err != nil {
		return "", fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(tmpOut.Name())
	tmpOut.Close()

	switch fileExt {
	case BPMN:
		cmdArgs = []string{"bpmn-to-image", fmt.Sprintf("%s:%s", tmpIn.Name(), tmpOut.Name()), "--scale", "0.7"}
	case DRAWIO:
		cmdArgs = []string{"drawio", "-x", "-f", outExt, "-o", tmpOut.Name(), tmpIn.Name()}
	case SVG:
		cmdArgs = []string{"inkscape", tmpIn.Name(), "--export-type=png", "--export-filename=" + tmpOut.Name()}
	default:
		return "", fmt.Errorf("unsupported file extension: %s", fileExt)
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s conversion failed: %w", fileExt, err)
	}

	imgData, err := os.ReadFile(tmpOut.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %w", err)
	}

	return base64.StdEncoding.EncodeToString(imgData), nil
}
