FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-node ./go-chain
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-miner ./go-chain/tools/miner
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-pool ./go-chain/tools/pool

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /tender-node .
COPY --from=builder /tender-miner ./tender-miner
COPY --from=builder /tender-pool ./tender-pool
RUN mkdir -p /root/data /root/certs /root/logs /root/pool
EXPOSE 8080 3030 9090 8083
ENV TZ=UTC
ENTRYPOINT ["/root/tender-node"]

