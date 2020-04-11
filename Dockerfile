FROM golang:1.12

WORKDIR /src

COPY . .

RUN go build -o capturereq cmd/*

EXPOSE 5000

ENTRYPOINT ["./capturereq"]
