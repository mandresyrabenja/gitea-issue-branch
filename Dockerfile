########################
# — Build stage —
########################
FROM golang:1.23.4-alpine AS builder

# git is needed to fetch modules and root CAs
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy *both* dependency files first so Docker-layer caching works
COPY go.mod go.sum ./
RUN go mod download

# Bring in the rest of the source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/action ./main.go

########################
# — Runtime stage —
########################
FROM alpine:3.20
RUN apk add --no-cache git
COPY --from=builder /out/action /usr/local/bin/action
ENTRYPOINT ["/usr/local/bin/action"]
