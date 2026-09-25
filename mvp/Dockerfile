FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /hearthmesh ./cmd/hearthmesh

FROM alpine:3.20
RUN adduser -D -u 10001 hearthmesh
RUN mkdir /data && chown hearthmesh:hearthmesh /data
USER hearthmesh
COPY --from=build /hearthmesh /usr/local/bin/hearthmesh
EXPOSE 8080
ENTRYPOINT ["hearthmesh"]
