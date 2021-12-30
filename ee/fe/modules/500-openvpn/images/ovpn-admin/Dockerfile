ARG BASE_ALPINE
ARG BASE_NODE_16_ALPINE
ARG BASE_GOLANG_17_ALPINE

FROM $BASE_ALPINE as src
WORKDIR /app
COPY deckhouse.patch /
RUN apk update && apk add git patch && git clone -b fixes https://github.com/vitaliy-sn/ovpn-admin.git . && patch -p1 < /deckhouse.patch
# rebuild 2

FROM node:14.2-alpine3.11 AS frontend
WORKDIR /app
COPY --from=src /app /app
RUN  cd frontend && npm install && npm run build

FROM $BASE_GOLANG_17_ALPINE AS backend
WORKDIR /app
COPY --from=src /app /app
RUN go build .

FROM $BASE_ALPINE
WORKDIR /app
RUN apk update && apk add bash openssl
COPY --from=backend /app/ovpn-admin /app/
COPY --from=frontend /app/frontend/static /app/frontend/static
COPY client.conf.tpl /app/templates/
COPY rootfs /
