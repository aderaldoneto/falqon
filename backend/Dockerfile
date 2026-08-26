FROM golang:1.25.7-alpine AS development

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

COPY . .

EXPOSE 8080

CMD ["go", "run", "./cmd/api"]
