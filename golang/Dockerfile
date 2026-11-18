# syntax=docker/dockerfile:1

FROM node:22.23.5-alpine AS assets
WORKDIR /app
COPY package.json ./
RUN npm install
COPY resources ./resources
COPY internal/web/templates ./internal/web/templates
RUN npm run build

FROM golang:1.24.3-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY --from=assets /app/internal/web/static ./internal/web/static
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/taskflow ./cmd/taskflow

FROM alpine:3.21.3 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/taskflow /usr/local/bin/taskflow
USER appuser
EXPOSE 8080
ENTRYPOINT ["taskflow"]
CMD ["web"]
