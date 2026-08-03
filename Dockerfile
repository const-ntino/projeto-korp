FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/http-server ./cmd/http-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/http-server /http-server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/http-server"]
