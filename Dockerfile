FROM golang:1.23-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/printing-master .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/printing-master ./printing-master

ENV PORT=8080
ENV STORAGE_DIR=/app/storage
VOLUME ["/app/storage"]
EXPOSE 8080

ENTRYPOINT ["./printing-master"]
