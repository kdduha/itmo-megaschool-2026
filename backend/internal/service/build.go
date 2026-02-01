package service

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kdduha/itmo-megaschool-2026/backend/internal/models"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

func getUserPrompt(req *models.ExplainRequest) string {
	userPrompt := fmt.Sprintf(userPromptTemplate, req.FileName)
	if req.Prompt != "" {
		userPrompt = fmt.Sprintf("%s\nDetails and questions: %s", userPrompt, req.Prompt)
	}
	return userPrompt
}

func (e *ExplainService) buildOpenAIReq(req *models.ExplainRequest) (*openai.ChatCompletionNewParams, error) {
	e.logger.Printf("start preprocessing file: %s\n", req.FileName)
	defer e.logger.Printf("finish preprocessing file: %s\n", req.FileName)

	var (
		messages []openai.ChatCompletionMessageParamUnion
		err      error
	)
	switch req.FileFormat {
	case PNG, JPEG, JPG:
		messages = e.buildImageMessages(req)
	case DRAWIO, BPMN:
		messages, err = e.buildDiagramMessages(req)
		if err != nil {
			return nil, fmt.Errorf("failed to convert drawio: %v", err)
		}
	case TXT:
		messages, err = e.buildTxtMessages(req)
		if err != nil {
			return nil, fmt.Errorf("failed to convert txt: %v", err)
		}
	default:
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
