GO=go

VERSION := 0.5.1
BUILD := `git rev-parse --short HEAD`
IMAGE := grapery-app:$(VERSION)-$(BUILD)
TARGETS := grapes

LDFLAGS += -X "$(project)/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "$(project)/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(project)/version.Version=$(VERSION)"
LDFLAGS += -X "$(project)/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
project=github.com/grapery/grapery


$(TARGETS): 
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-app  $(project)/app/grapes/
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-worker  $(project)/app/syncworker/

withpgo: $(TARGETS)
	$(GO) build  -pgo=./sample.pgo -ldflags  '$(LDFLAGS)' -o grapes-app  $(project)/app/grapes/
	$(GO) build  -pgo=./sample.pgo -ldflags  '$(LDFLAGS)' -o grapes-worker  $(project)/app/syncworker/

image: $(TARGETS)
	tar cvf build.tar $(TARGETS)-app
	docker build -f dockerfiles/Dockerfile -t $(IMAGE) .
	rm -f build.tar 
	@echo "image: $(IMAGE)"

check:
	@$(GO) tool vet ${SRC}

test:
	@$(GO) test -race `$(GO) list ./... 

clean:
	rm -f $(TARGETS)

cov:
	gocov test -timeout=20m -race -v `$(GO) list ./... |egrep -v "app"`

cert:
	sh ./certs/gen.sh

	