GO?=		go
INSTALL?=	install

PREFIX?=	/usr/local
BINDIR=		${PREFIX}/bin

CLIENT=		client
SERVER=		server

CLIENT_PKG=	./cmd/client
SERVER_PKG=	./cmd/server

SERVER_HOST?=	localhost
SERVER_PORT?=	443
WG_TARGET?=	127.0.0.1:51820

PROXY_TEMPLATE=	https://${SERVER_HOST}:${SERVER_PORT}/masque?h={target_host}&p={target_port}

.PHONY: all clean install uninstall fmt test run-server-masque run-client-masque

all: ${CLIENT} ${SERVER}

${CLIENT}:
	${GO} build -o ${.TARGET} ${CLIENT_PKG}

${SERVER}:
	${GO} build -o ${.TARGET} ${SERVER_PKG}

fmt:
	${GO} fmt ./...

test:
	${GO} test ./...

clean:
	rm -f ${CLIENT} ${SERVER}

install: all
	mkdir -p ${DESTDIR}${BINDIR}
	${INSTALL} -m 755 ${CLIENT} ${DESTDIR}${BINDIR}/${CLIENT}
	${INSTALL} -m 755 ${SERVER} ${DESTDIR}${BINDIR}/${SERVER}

uninstall:
	rm -f ${DESTDIR}${BINDIR}/${CLIENT}
	rm -f ${DESTDIR}${BINDIR}/${SERVER}

run-server-masque: ${SERVER}
	./${SERVER} \
		-mode masque \
		-listen :${SERVER_PORT} \
		-target ${WG_TARGET} \
		-proxy-template '${PROXY_TEMPLATE}'

run-client-masque: ${CLIENT}
	./${CLIENT} \
		-mode masque \
		-listen 127.0.0.1:51821 \
		-target ${WG_TARGET} \
		-proxy-template '${PROXY_TEMPLATE}' \
		-server-name localhost \
		-insecure
