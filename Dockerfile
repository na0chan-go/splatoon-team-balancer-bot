FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bot ./cmd/bot

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
RUN chown app:app /app

COPY --from=builder /app/bot /app/bot

USER app

ENTRYPOINT ["/app/bot"]
