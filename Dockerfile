FROM golang:alpine as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /app/output

FROM alpine

EXPOSE 80

WORKDIR /app

COPY --from=builder /app/output .

CMD [ "/app/output" ]
