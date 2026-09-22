FROM golang:1.27-alpine AS builder

WORKDIR /build

# Кэшируем зависимости отдельным слоем: пересобираются только при
# изменении go.mod/go.sum, а не при каждой правке main.go
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 - статическая линковка, бинарник не тянет зависимостей
# от libc внутри alpine-образа
RUN CGO_ENABLED=0 GOOS=linux go build -o /discovery src/main.go

# ---- Стадия 2: минимальный runtime-образ ----
FROM alpine:3.20

# iproute2 и iputils нужны только для диагностики внутри контейнера
# (ip addr, ip route, ping) - удобно при отладке multicast на защите
RUN apk add --no-cache iproute2 iputils tcpdump

COPY --from=builder /discovery /usr/local/bin/discovery

ENTRYPOINT ["/usr/local/bin/discovery"]