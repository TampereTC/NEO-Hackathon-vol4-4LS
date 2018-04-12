FROM golang:1.8

# ENV https_proxy http://10.144.1.10:8080

RUN go get -d -v "github.com/wcharczuk/go-chart"

WORKDIR /go/src/app
COPY . .

CMD [ "go", "run", "./main.go" ]
