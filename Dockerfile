FROM golang:1.24 AS go-builder
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -v -o /usr/local/bin/main ./cmd/rangedb
RUN chmod +x /usr/local/bin/main

# This produces a very tiny image with only the binary and no dependencies.
FROM scratch
COPY --from=go-builder /usr/local/bin/main /usr/local/bin/main
USER 1000:1000
WORKDIR /data
ENTRYPOINT ["/usr/local/bin/main"]