FROM golang:1.25-bookworm AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o server cmd/main.go

FROM rlespinasse/drawio-desktop-headless:v1.52.0

COPY --from=build /app/server /usr/local/bin/server

# drawio alias entrypoint.sh
RUN ln -s /opt/drawio-desktop/entrypoint.sh /usr/local/bin/drawio

RUN apt-get update && \
    apt-get install -y inkscape nodejs npm mupdf mupdf-tools libmupdf-dev libffi8 && \
    apt-get install -y nodejs npm && \
    npm install -g bpmn-to-image@latest --omit=dev && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m appuser
USER appuser

RUN CI=yes npx puppeteer browsers install chrome

ENTRYPOINT ["server"]
CMD []
