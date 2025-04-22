FROM golang:1.23 AS builder
WORKDIR /app
COPY chatroom-go/go.mod chatroom-go/go.sum ./
RUN go mod download
COPY chatroom-go/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]

