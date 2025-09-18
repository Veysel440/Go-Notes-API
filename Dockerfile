# build
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/api ./cmd/api

# run
FROM gcr.io/distroless/base-debian12
ENV APP_ENV=prod APP_PORT=8080
WORKDIR /app
COPY --from=build /app/api /app/api
COPY migrations /app/migrations
COPY .env.example /app/.env.example
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]