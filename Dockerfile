# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app
COPY . .
WORKDIR /app/cmd/vibeaura
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /vibeaura .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates git

COPY --from=builder /vibeaura /usr/local/bin/vibeaura

RUN mkdir -p /run/agentic /root/.vibeauracle
ENV AGENTIC_RUN_DIR=/run/agentic

ENTRYPOINT ["vibeaura"]
CMD ["daemon", "start"]
