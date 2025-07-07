# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/action ./main.go

# Runtime Stage
FROM alpine:3.20
COPY --from=builder /out/action /usr/local/bin/action
ENTRYPOINT ["/usr/local/bin/action"]
