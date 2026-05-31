ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config libgl1-mesa-dev libxrandr-dev libxcursor-dev \
    libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -o /build/otaclock2 .

FROM debian:bookworm-slim AS runtime

LABEL org.opencontainers.image.title="otaclock" \
      org.opencontainers.image.description="Metal Gear Solid 2-style Otacon desktop clock" \
      org.opencontainers.image.authors="Austin Millan" \
      org.opencontainers.image.vendor="Austin Millan" \
      org.opencontainers.image.url="https://gitlab.com/austin-millan/otaclock" \
      org.opencontainers.image.documentation="https://gitlab.com/austin-millan/otaclock/-/blob/main/README.md" \
      org.opencontainers.image.source="https://gitlab.com/austin-millan/otaclock"

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libgl1-mesa-dri libx11-6 libxrandr2 libxcursor1 libxinerama1 libxi6 libasound2 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /build/otaclock2 /app/otaclock2

ENV DISPLAY=:0

# GUI: docker run --rm -e DISPLAY -v /tmp/.X11-unix:/tmp/.X11-unix otaclock2:local
ENTRYPOINT ["/app/otaclock2"]
