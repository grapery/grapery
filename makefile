GO=go

VERSION := 0.0.1
BUILD := `git rev-parse --short HEAD`
IMAGE := grapery-app:$(VERSION)-$(BUILD)
TARGETS := grapes-app

LDFLAGS += -X "$(project)/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "$(project)/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(project)/version.Version=$(VERSION)"
LDFLAGS += -X "$(project)/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
project=$(GOPATH)/src/github.com/grapery/grapery


$(TARGETS): 
	$(GO) build -o grapes-app -ldflags  '$(LDFLAGS)' $(project)/app/grapes.go

image: $(TARGETS)
	tar cvf build.tar $(TARGETS)
	docker build -f dockerfiles/Dockerfile -t $(IMAGE) .
	rm -f build.tar 
	@echo "image: $(IMAGE)"

check:
	@$(GO) tool vet ${SRC}

test:
	@$(GO) test -race `$(GO) list ./... | grep -v /vendor/`

clean:
	rm -f $(TARGETS)

cov: 
	gocov test -timeout=20m -race -v `$(GO) list ./... |egrep -v "app"` 