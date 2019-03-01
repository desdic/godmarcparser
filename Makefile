.PHONY: all test docker build clean packages
default: all

GOC=go
VERSION=1.0.3
REVISION=$(shell git rev-parse --short HEAD)
BUILDDATE=$(shell date +'%Y%m%d')
FLAGS=-ldflags="-X github.com/desdic/godmarcparser/version.version=${VERSION}-${REVISION} -X github.com/desdic/godmarcparser/version.builddate=${BUILDDATE}"
PACKAGE = godmarcparser

all: clean build

build: ${PACKAGE}

test:
	CGO_ENABLED=1 $(GOC) test -race $(FLAGS) ./...

cover:
	CGO_ENABLED=1 $(GOC) test -cover $(FLAGS) ./...

${PACKAGE}:
	$(GOC) build $(FLAGS) -o bin/$@

docker:
	docker build -f docker/dockerfile -t godmarc:$(VERSION) .

packages:
	rm -rf artifacts
	mkdir -p packages/ubuntu/xenial
	docker build -f docker/dockerfile.xenial -t godmarc:xenial .
	mkdir artifacts
	docker run --rm -iv${PWD}/artifacts:/host-volume godmarc:xenial sh -c "cp /home/godmarcreport_${VERSION}_amd64.changes /host-volume/ && cp /home/godmarcreport_${VERSION}_amd64.deb /host-volume/" 
	docker rmi godmarc:xenial
	mv artifacts/* packages/ubuntu/xenial/
	sudo chown -R $(shell id -u):$(shell id -g) packages
	rm -rf artifacts

install:
	install --directory --owner=root --group=root --mode=755 $(DESTDIR)/usr/sbin
	install --directory --owner=root --group=root --mode=755 $(DESTDIR)/usr/share/${PACKAGE}
	install --directory --owner=root --group=root --mode=755 $(DESTDIR)/usr/share
	install --directory --owner=root --group=root --mode=755 $(DESTDIR)/etc/${PACKAGE}
	install --owner=root --group=root --mode=644 etc/config.json $(DESTDIR)/etc/${PACKAGE}/config.json
	install --owner=root --group=root --mode=755 bin/${PACKAGE} $(DESTDIR)/usr/sbin/${PACKAGE}
	install --owner=root --group=root --mode=644 README.md $(DESTDIR)/usr/share/${PACKAGE}/README.md
	install --directory --owner=root --group=root --mode=755 "${DESTDIR}/lib/systemd/system"
	install --owner=root --group=root --mode=644 etc/systemd/${PACKAGE}.service "${DESTDIR}/lib/systemd/system/${PACKAGE}.service"

clean:
	rm -f bin/${PACKAGE}
	rm -rf pkg

