FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mafia-bot ./main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/mafia-bot .
COPY --from=builder /app/webapp/ ./webapp/
EXPOSE 8080
CMD ["./mafia-bot"]
