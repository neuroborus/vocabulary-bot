# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-s -w" \
	-o /out/vocabulary-bot \
	./cmd/vocabulary-bot

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S app \
	&& adduser -S -G app -h /home/app app

WORKDIR /home/app

COPY --from=build /out/vocabulary-bot /usr/local/bin/vocabulary-bot

USER app:app

ENV APP_ENV=production

ENTRYPOINT ["/usr/local/bin/vocabulary-bot"]
