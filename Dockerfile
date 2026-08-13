# syntax=docker/dockerfile:1

# ---- Stage 1: build the React frontend ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
# Install deps first (better layer caching)
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
# Build
COPY frontend/ ./
RUN npm run build          # -> /app/frontend/dist

# ---- Stage 2: build the Go binary ----
FROM golang:1.26-alpine AS backend
WORKDIR /src
# Module cache layer
COPY go.mod go.sum ./
RUN go mod download
# Sources
COPY . .
# Static binary (no libc dependency) for the production web entrypoint
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/web ./cmd/web

# ---- Stage 3: minimal runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=backend /out/web ./web
COPY --from=frontend /app/frontend/dist ./frontend/dist
ENV PORT=8080 \
    STATIC_DIR=./frontend/dist \
    GIN_MODE=release
EXPOSE 8080
USER app
CMD ["./web"]
