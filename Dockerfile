FROM golang:latest

RUN apt-get -y update
RUN apt-get install -y ffmpeg

EXPOSE 8080

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go mod download
RUN go build -o main .

CMD ["/app/main"]
