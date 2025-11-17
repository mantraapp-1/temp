FROM golang:1.22-bookworm

# Install Python, pip, ffmpeg and basic deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip ffmpeg git \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Go deps first (better layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project
COPY . .

# Python deps (CPU; you can tweak versions or use a different whisper backend)
# NOTE: whisper-large-v3 is big; on CPU it'll be slow. Change model in the script if needed.
RUN pip3 install --no-cache-dir \
    torch \
    torchaudio \
    transformers \
    librosa

# Build the Go server (worker + HTTP)
RUN go build -o server ./cmd/server

EXPOSE 8080

CMD ["./server"]
