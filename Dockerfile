FROM golang:latest
FROM jrottenberg/ffmpeg:latest

EXPOSE 8080

RUN mkdir /app
ADD . /app
WORKDIR /app

# RUN go mod download
RUN go mod tidy && go mod vendor
RUN go build -o main .

CMD ["/app/main"]
