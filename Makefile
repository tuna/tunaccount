LDFLAGS="-X main.buildstamp=`date -u '+%s'` -X main.githash=`git rev-parse HEAD`"
ARCH ?= linux-amd64
ARCH_LIST = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(ARCH_LIST))
GOARCH = $(word 2, $(ARCH_LIST))
BUILDBIN = tunaccount

all: $(BUILDBIN)

build-$(ARCH):
	mkdir -p $@

$(BUILDBIN): % : build-$(ARCH) build-$(ARCH)/%

$(BUILDBIN:%=build-$(ARCH)/%) : build-$(ARCH)/% : main.go 
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go get ./
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ -ldflags ${LDFLAGS} github.com/tuna/tunaccount

