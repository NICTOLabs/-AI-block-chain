FROM golang:1.22-alpine

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o tender-node .

EXPOSE 8080 3030

CMD ["./tender-node"]
