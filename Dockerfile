FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o broker .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/broker /broker
EXPOSE 8080
ENTRYPOINT ["/broker"]
