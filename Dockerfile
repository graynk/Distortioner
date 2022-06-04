FROM golang:1.18-bullseye as build
WORKDIR /go/src/distortioner
COPY app .
RUN go test ./...
RUN go build

FROM ghcr.io/graynk/ffmpegim as release

WORKDIR app
COPY --from=build /go/src/distortioner/distortioner distortioner

ENTRYPOINT ["./distortioner"]