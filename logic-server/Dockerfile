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

# Copy logic-server module
WORKDIR /build/logic-server
COPY logic-server/go.mod logic-server/go.sum ./
RUN go mod download

# Copy source code
WORKDIR /build
COPY common/ ./common/
COPY logic-server/ ./logic-server/

# Build
WORKDIR /build/logic-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/logic-server .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/logic-server .
COPY logic-server/config.yaml .

EXPOSE 9090

ENTRYPOINT ["./logic-server"]
