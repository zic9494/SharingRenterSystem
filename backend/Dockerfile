# build
FROM golang:1.22-alpine AS build 
# Go 1.22 的精簡版 Alpine Linux 映像檔
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./

# run
FROM alpine:3.20
WORKDIR /app
ENV PORT=8080 DB_PATH=/data/app.db APP_ENV=prod
RUN adduser -D appuser
USER appuser
COPY --from=build /app/server /app/server
# 掛載到8080端口
EXPOSE 8080
CMD [ "/app/server" ]