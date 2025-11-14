FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pr-reviewer ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/pr-reviewer ./pr-reviewer
COPY --from=builder /app/db/migrations ./db/migrations
ENV HTTP_ADDR=:8080
ENV DB_URL=postgres://postgres:postgres@db:5432/pr_review?sslmode=disable
ENV ADMIN_TOKEN=admin-secret
ENV USER_TOKEN=user-secret
ENV RUN_MIGRATIONS=true
EXPOSE 8080
ENTRYPOINT ["./pr-reviewer"]
