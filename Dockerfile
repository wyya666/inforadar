ARG GO_IMAGE=golang:1.26-alpine
ARG NODE_IMAGE=node:24-alpine
ARG API_IMAGE=alpine:3.23
ARG NGINX_IMAGE=nginx:stable-alpine

FROM ${GO_IMAGE} AS backend-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY config.example.yaml ./
RUN go test ./... && go vet ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/inforadar ./cmd/server

FROM ${NODE_IMAGE} AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM ${API_IMAGE} AS api
RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 10001 inforadar && \
    adduser -S -D -H -u 10001 -G inforadar inforadar
WORKDIR /app
COPY --from=backend-build /out/inforadar /usr/local/bin/inforadar
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/inforadar"]

FROM ${NGINX_IMAGE} AS web
COPY --from=frontend-build /src/frontend/dist /usr/share/nginx/html
COPY deploy/nginx.conf.template /etc/nginx/templates/default.conf.template
EXPOSE 80
