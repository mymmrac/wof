FROM golang:1.26.4-alpine AS build

RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GO111MODULE=on go build -ldflags="-s -w" -installsuffix "static" -trimpath -o /bin/wof .

FROM scratch AS release

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/wof /wof

ENTRYPOINT ["/wof"]
