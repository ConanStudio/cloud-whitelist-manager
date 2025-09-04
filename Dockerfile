# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# China Go module proxy examples (commented by default). Uncomment one line to enable when building in China.
# Recommended mirrors:
#  - https://goproxy.cn
#  - https://mirrors.aliyun.com/goproxy/ (Alibaba mirror)
# Example: uncomment the ENV line below to set GOPROXY inside the builder image.
# ENV GOPROXY=https://goproxy.cn,direct
# Alternative (Alibaba):
# ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cloud-whitelist-manager ./cmd/cloud-whitelist-manager

# Final stage
FROM alpine:latest

# China Alpine APK mirror examples (commented by default). Uncomment or pass a build-arg to use a domestic mirror when building in China.
# Recommended mirrors:
#  - https://mirrors.tuna.tsinghua.edu.cn/alpine (TUNA/Tsinghua)
#  - https://mirrors.aliyun.com/alpine (Alibaba)
#
# Two options to use a mirror:
# 1) Replace /etc/apk/repositories inside the image (uncomment to enable):
#    RUN echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.18/main" > /etc/apk/repositories \
#        && echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.18/community" >> /etc/apk/repositories \
#        && apk --no-cache add ca-certificates
#
# 2) Use a build-arg to set a mirror at build time (example):
#    docker build --build-arg APK_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/alpine -t $(BINARY_NAME) .
#  And in Dockerfile you could (optionally) read it like:
#    ARG APK_MIRROR
#    RUN if [ -n "$APK_MIRROR" ]; then \
#          echo "$APK_MIRROR/v3.18/main" > /etc/apk/repositories; \
#          echo "$APK_MIRROR/v3.18/community" >> /etc/apk/repositories; \
#        fi && apk --no-cache add ca-certificates
#
# The actual default behavior (no mirror) is preserved below.
# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/cloud-whitelist-manager .

# Copy default configuration
COPY config.yaml ./config.yaml

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose no ports as this is a background service

# Command to run the application
ENTRYPOINT ["./cloud-whitelist-manager"]
CMD ["--config", "config.yaml"]