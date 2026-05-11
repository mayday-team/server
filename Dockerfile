FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/mayday-server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/mayday-server /app/mayday-server
COPY migrations /app/migrations
EXPOSE 3001
USER 65532:65532
ENTRYPOINT ["/app/mayday-server"]
