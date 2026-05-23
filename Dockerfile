# Stage 1: build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: runtime with Python + PyMuPDF
FROM python:3.11-slim
WORKDIR /app

COPY parser/requirements.txt ./parser/requirements.txt
RUN pip install --no-cache-dir -r parser/requirements.txt

COPY --from=builder /app/server .
COPY parser/ ./parser/
COPY web/   ./web/

RUN mkdir -p /data /storage/pdfs

ENV PORT=8080 \
    DATA_DIR=/data \
    STORAGE_DIR=/storage \
    PYTHON_BIN=python3 \
    PARSER_SCRIPT=/app/parser/extract.py

EXPOSE 8080
CMD ["./server"]
