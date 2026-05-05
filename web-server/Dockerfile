FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

# Set GOPROXY environment variable
ENV GOPROXY=https://goproxy.cn,direct
RUN go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /build

# Copy common module first
WORKDIR /build/common
COPY common/go.mod common/go.sum ./
RUN go mod download

# Copy web-server module
WORKDIR /build/web-server
COPY web-server/go.mod web-server/go.sum ./
RUN go mod download

# Copy source code
WORKDIR /build
COPY common/ ./common/
COPY web-server/ ./web-server/

# Build
WORKDIR /build/web-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/web-server .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/web-server .
COPY web-server/config.yaml .

EXPOSE 8080

ENTRYPOINT ["./web-server"]
