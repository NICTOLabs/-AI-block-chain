FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-node .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /tender-node .
RUN mkdir -p /root/data
EXPOSE 10000 3030
ENV TZ=UTC
ENTRYPOINT ["/root/tender-node"]
