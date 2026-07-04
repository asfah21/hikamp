# ==========================================
# STAGE 1: Build biner Golang
# ==========================================
FROM golang:1.25-alpine AS builder

# Install git dan ca-certificates (diperlukan untuk kirim email/webhook HTTPS)
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

# Copy file dependency dulu agar memanfaatkan caching Docker
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code (termasuk folder templates dan assets)
COPY . .

# Kompilasi Go menjadi biner tunggal yang statis dan optimal untuk Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

# ==========================================
# STAGE 2: Runtime Environment
# ==========================================
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy biner hasil compile dari Stage 1
COPY --from=builder /app/main .

# COPY FOLDER STATIS
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/assets ./assets

# Ekspos port sesuai aplikasi Gin (misal 8080)
EXPOSE 8080

# Jalankan aplikasi
CMD ["./main"]
