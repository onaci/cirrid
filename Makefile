
COMMIT ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo v0)
BUILDTIME ?= $(shell date +"%Y.%m.%d.%H%M%S")

BUILDSETTING=-X github.com/onaci/cirrid/install.Commit=$(COMMIT) -X github.com/onaci/cirrid/install.BuildTime=$(BUILDTIME)

all: clean cirrid cirrid-osx cirrid.exe

clean:
	rm cirrid cirrid-osx cirrid.exe | true

cirrid:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static" $(BUILDSETTING)' -o cirrid

cirrid-osx:
	env GOOS=darwin GOARCH=amd64 go build -a -o cirrid-osx -ldflags="-s -w $(BUILDSETTING)"

cirrid.exe:
	env GOOS=windows GOARCH=amd64 go build -a -o cirrid.exe -ldflags="-s -w $(BUILDSETTING)"

github-release: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static" $(BUILDSETTING)' -o cirrid
	env GOOS=darwin GOARCH=amd64 go build -a -o cirrid-osx -ldflags="-s -w $(BUILDSETTING)"
	# need to fix windows builds later
	#env GOOS=windows GOARCH=amd64 go build -a -o cirrid.exe -ldflags="-s -w $(BUILDSETTING)"
	#		--attach "cirrid.exe#MS Windows amd64"


	hub release create \
			--draft \
			--prerelease \
			--commitish master \
			--message "02 March 2021, cirrid logs -f, and ssh speedups." \
			--attach "cirrid#Linux amd64" \
			--attach "cirrid-osx#OS X amd64" \
			v0.$(BUILDTIME)

docker-release:
	docker build -t onaci/cirrid:cmdline .
	docker push onaci/cirrid:cmdline
