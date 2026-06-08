FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mafia-bot ./...

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=build /app/mafia-bot .
COPY --from=build /app/webapp ./webapp
EXPOSE 8080
CMD ["./mafia-bot"]
