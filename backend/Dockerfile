FROM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY config/ config/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/api

FROM ubuntu:22.04

# Install Python3, pip, ffmpeg and other necessary packages
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the main binary and config
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

# Copy Python scripts and requirements
COPY utils/ ./utils/
COPY process_song.py ./
COPY requirements.txt .

# Create necessary directories
RUN mkdir -p uploads output data_roots

# Install Python dependencies
RUN python3 -m pip install -r requirements.txt

EXPOSE 8000

CMD ["./main"]