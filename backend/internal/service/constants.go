package service

const (
	PNG    = "png"
	JPEG   = "jpeg"
	JPG    = "jpg"
	SVG    = "svg"
	DRAWIO = "drawio"
	BPMN   = "bpmn"
	TXT    = "txt"
	PDF    = "pdf"
)

const (
	systemPromptImage = `
You are an assistant. Explain the uploaded diagram briefly and clearly.
User can give extra information or ask certain questions about the diagram.`

	userPromptTemplate = "Filename: %s"
)
