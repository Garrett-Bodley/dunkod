FROM golang:1.24-alpine AS builder


RUN apk update
RUN apk add --no-cache git gcc g++ musl-dev
WORKDIR /app
ENV CGO_ENABLED=1
RUN go env -w GOCACHE=/go-cache
RUN go env -w GOMODCACHE=/gomod-cache
COPY go.* .
RUN --mount=type=cache,target=/gomod-cache go mod download
COPY . .
RUN --mount=type=cache,target=/gomod-cache \
  --mount=type=cache,target=/go-cache \
  go build .

FROM alpine:latest
RUN apk update
RUN apk add --no-cache ffmpeg ca-certificates tzdata
RUN apk add --no-cache sqlite
WORKDIR /app
COPY --from=builder /app/dunkod .
COPY --from=builder /app/db/migrations ./db/migrations
COPY --from=builder /app/views ./views
COPY --from=builder /app/static ./static

CMD ["sh", "-c", "./dunkod -p 2>&1 | tee -a ../logs/log.txt"]