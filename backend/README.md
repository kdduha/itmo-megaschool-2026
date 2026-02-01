# Backend

Here you can find a backend proxy service, that:
- converts diagram files (`bpmn`, `drawio`) to images
- streams diagrams for an explanation to OpenAI-like backends
- caches OpenAI backend responses with Redis

You can find swagger here: `http://localhost:8080/swagger/index.html#/`

## Local Setup

You can check the basic backend configuration and available params [here](./internal/config/config.go)

1. Install bpmn diagram converter [`bpmn-to-image`](https://github.com/bpmn-io/bpmn-to-image)
1. Install [`drawio`](https://github.com/jgraph/drawio) desktop app
1. Start the server:
   ```sh
   go run cmd/main.go
   ```

## Docker Setup

You can set envs in order to configure the server. Be aware of the image size, cause it's heavy due to 
environment setup of diagram convert CLIs

1. Build an image:
   ```sh
   docker build --platform linux/amd64 -t go-backend .
   ```
1. Start the server:
   ```sh
   docker run -it --rm go-backend
   ```

## Usage Examples

- No-stream requests
```sh
curl -X POST http://localhost:8080/explain \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Explain what you see",
    "file_base64": "'"$(base64 -i <your_diagram>.png)"'",
    "file_name": "<your_diagram>.png",
    "file_format": "png"
  }'
```

- Stream requests
```sh
curl -N -X POST http://localhost:8080/explain/stream \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Explain the diagram",
    "file_base64": "'"$(base64 -i <your_diagram>.png)"'",
    "file_name": "<your_diagram>.png",
    "file_format": "png"
  }'
```

## Developing

Some useful commands:
- Update swagger doc for new handlers
  ```sh
  swag init -g cmd/main.go -o docs
  ```

- Code formatting
  ```sh
  goimports -w . && gofmt -w .
  ```
