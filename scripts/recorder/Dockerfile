# CLI Recording Environment
# Produces GIFs from terminal sessions (supports TUI apps)

FROM golang:1.25-bookworm AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /alab ./cmd/alab

# Recording environment
FROM debian:bookworm-slim

# Install recording tools and fonts
RUN apt-get update && apt-get install -y --no-install-recommends \
    asciinema \
    expect \
    tree \
    curl \
    ca-certificates \
    fonts-dejavu-core \
    && rm -rf /var/lib/apt/lists/*

# Install agg (asciinema GIF generator) - download pre-built binary
RUN curl -L https://github.com/asciinema/agg/releases/download/v1.5.0/agg-x86_64-unknown-linux-gnu -o /usr/local/bin/agg \
    && chmod +x /usr/local/bin/agg

# Copy built binary
COPY --from=builder /alab /usr/local/bin/alab

# Set up recording environment
WORKDIR /workspace
ENV TERM=xterm-256color
ENV HOME=/workspace

# Create output directory and bashrc with clean prompt
RUN mkdir -p /output /workspace/project && \
    echo "PS1='demo\$ '" > /workspace/.bashrc && \
    echo "cd /workspace/project" >> /workspace/.bashrc

# Copy recording scripts
COPY scripts/recorder/record.sh /usr/local/bin/record
COPY scripts/recorder/scenarios/*.sh /scenarios/
RUN chmod +x /usr/local/bin/record /scenarios/*.sh

ENTRYPOINT ["/usr/local/bin/record"]
