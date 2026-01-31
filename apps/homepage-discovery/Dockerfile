FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy all source files
COPY . .

# Download dependencies and build
RUN go mod download || true
RUN CGO_ENABLED=0 GOOS=linux go build -o homepage-discovery .

# Runtime image - use scratch for minimal image, no packages needed
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/homepage-discovery /homepage-discovery

USER 65534:65534

ENTRYPOINT ["/homepage-discovery"]
