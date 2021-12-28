ARG BASE_PYTHON_ALPINE
ARG BASE_NODE_16_ALPINE
ARG BASE_GOLANG_17_ALPINE

FROM $BASE_PYTHON_ALPINE as src
WORKDIR /app
RUN apk update && apk add git && git clone -b 1.7.5 https://github.com/flant/ovpn-admin.git .

FROM node:14.2-alpine3.11 AS frontend
WORKDIR /app
COPY --from=src /app /app
RUN  cd frontend && npm install && npm run build

FROM $BASE_GOLANG_17_ALPINE AS backend
WORKDIR /app
COPY --from=src /app /app
RUN go build .

FROM $BASE_PYTHON_ALPINE
WORKDIR /app
COPY --from=backend /app/ovpn-admin /app
COPY --from=frontend /app/frontend/static /app/frontend/static
