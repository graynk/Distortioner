FROM debian:buster-slim as release

RUN mkdir app
WORKDIR app

RUN apt-get -y update && \
    apt-get -y upgrade && \
    # would be nice to compile ffmpeg with only needed stuff as well, but whatever
    apt-get install -y ffmpeg \
    # honestly stolen and modified from https://github.com/dooman87/imagemagick-docker/blob/master/Dockerfile.buster
    git make gcc pkg-config autoconf curl g++ \
    # IM
    libpng16-16 libpng-dev libjpeg62-turbo libjpeg62-turbo-dev libglib2.0-dev liblqr-1-0 liblqr-1-0-dev libwebp6 libwebp-dev libwebpmux3 libwebpdemux2 libgomp1 && \
    # Building ImageMagick
    git clone --depth 1 https://github.com/ImageMagick/ImageMagick.git && \
    cd ImageMagick && \
    ./configure --without-magick-plus-plus --disable-docs --disable-static && \
    make && make install && \
    ldconfig /usr/local/lib && \
    apt-get remove --autoremove --purge -y gcc make cmake curl g++ yasm git autoconf pkg-config libpng-dev libjpeg62-turbo-dev libglib2.0-dev liblqr-1-0-dev libwebp-dev && \
    rm -rf /var/lib/apt/lists/* && \
    rm -rf /ImageMagick

FROM golang as build
WORKDIR /go/src/distortioner
COPY . .
RUN go build

FROM release
COPY --from=build /go/src/distortioner/distortioner distortioner
ENTRYPOINT ["./distortioner"]