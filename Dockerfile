# Build the React/Vite console separately, then embed it in the Go API binary.
FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.3-alpine3.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Remove stale local embed assets so only this Vite build is embedded.
RUN rm -rf ./internal/api/v1/static/*
COPY --from=web /src/frontend/dist/ ./internal/api/v1/static/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cyberos ./cmd/cyberos

FROM alpine:3.24.4
RUN apk add --no-cache ca-certificates tzdata nmap curl \
    && addgroup -S -g 10001 cyberos \
    && adduser -S -D -H -u 10001 -G cyberos cyberos
WORKDIR /app
COPY --from=build --chown=10001:10001 /out/cyberos /app/cyberos
COPY --from=build --chown=10001:10001 /src/migrations /app/migrations
USER 10001:10001
ENV CYBEROS_ENV=production
ENV CYBEROS_API_ADDR=:8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/api/v1/ready || exit 1
ENTRYPOINT ["/app/cyberos", "serve"]
