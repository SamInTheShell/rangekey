# Multi-stage build for RangeDB
FROM golang:1.24-bookworm AS builder

# Set working directory
WORKDIR /build

# Copy go mod files first for layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary as a static binary with no external dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-w -s" -o rangedb ./cmd/rangedb

# Runtime stage using distroless
FROM gcr.io/distroless/static:nonroot

# Copy binary from builder
COPY --from=builder /build/rangedb /usr/local/bin/rangedb

# Set working directory
WORKDIR /data

# Expose ports
# 8080 - peer communication
# 8081 - client API
EXPOSE 8080 8081

# Default command
CMD ["/usr/local/bin/rangedb", "server", "--cluster-init"]