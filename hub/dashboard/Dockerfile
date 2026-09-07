FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /dashboard .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && mkdir -p /data
COPY --from=build /dashboard /dashboard
ENV LISTEN=:8090 DATA_FILE=/data/state.json
EXPOSE 8090
ENTRYPOINT ["/dashboard"]
