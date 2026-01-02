# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxy .

# Runtime stage - nginx:alpine base
FROM nginx:alpine

# Install minimal dependencies (gettext for envsubst)
RUN apk --no-cache add ca-certificates gettext

# Copy binary from builder
COPY --from=builder /build/proxy /usr/local/bin/proxy

# Copy nginx main config template (processed at runtime with envsubst)
COPY nginx/nginx.conf /etc/nginx/nginx.conf.template

# Create directories for generated configs
RUN mkdir -p /etc/nginx/conf.d && \
    touch /etc/nginx/conf.d/proxy.conf && \
    touch /etc/nginx/conf.d/http-proxy.conf

# Copy custom entrypoint
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Expose common ports
EXPOSE 80 443 53/udp

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["watch"]
