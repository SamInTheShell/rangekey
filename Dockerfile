FROM golang:1.24 AS go-builder
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -v -o /usr/local/bin/rangedb ./cmd/rangedb
RUN chmod +x /usr/local/bin/rangedb

# This produces a very tiny image with only the binary and no dependencies.
FROM scratch
COPY --from=go-builder /usr/local/bin/rangedb /usr/local/bin/rangedb
USER 1000:1000
WORKDIR /data
ENTRYPOINT ["/usr/local/bin/rangedb"]
