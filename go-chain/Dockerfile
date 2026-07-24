FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go-chain/go.mod go-chain/go.sum* ./
RUN cd go-chain && go mod download
COPY go-chain/ ./go-chain/
RUN cd go-chain && CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-node .
RUN cd go-chain && CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-miner ./tools/miner
RUN cd go-chain && CGO_ENABLED=0 go build -ldflags="-s -w" -o /tender-pool ./tools/pool

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
