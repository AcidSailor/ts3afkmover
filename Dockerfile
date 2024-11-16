FROM golang:1.23.2 AS build

COPY . .

RUN CGO_ENABLED=0 go build -o /app/ts3afkmover ./cmd/main.go

FROM scratch

WORKDIR /app

USER 1000

COPY --from=build /app/ts3afkmover .

ENTRYPOINT ["./ts3afkmover"]
