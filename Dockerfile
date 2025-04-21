FROM golang:1.23.8
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o ./server
EXPOSE 8080
CMD ["./server"]
