LDFLAGS="-X main.buildstamp=`date -u '+%s'` -X main.githash=`git rev-parse HEAD`"

all: tunaccount

build:
	mkdir -p build

tunaccount: build
	go build -o build/tunaccount -ldflags ${LDFLAGS} github.com/tuna/tunaccount

