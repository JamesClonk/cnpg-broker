# builder image
FROM golang:1.25-alpine AS builder
WORKDIR /go/src/github.com/JamesClonk/cnpg-broker
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o cnpg-broker .

# app image
FROM alpine:3.23
LABEL author="JamesClonk <jamesclonk@jamesclonk.ch>"

RUN apk --no-cache add ca-certificates

ENV PATH=$PATH:/app
WORKDIR /app

COPY public ./public/
COPY static ./static/
COPY --from=builder /go/src/github.com/JamesClonk/cnpg-broker/cnpg-broker ./cnpg-broker

EXPOSE 8080
CMD ["./cnpg-broker"]
