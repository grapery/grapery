GO=go

VERSION := 0.5.2
BUILD := `git rev-parse --short HEAD`
IMAGE := grapery-app:$(VERSION)-$(BUILD)
TARGETS := grapes asynctask

LDFLAGS += -X "$(project)/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "$(project)/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(project)/version.Version=$(VERSION)"
LDFLAGS += -X "$(project)/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
project=github.com/grapery/grapery


$(TARGETS): 
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-app  $(project)/app/grapes/
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-pay  $(project)/app/vippay/
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-llmchat  $(project)/app/llmchat/
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-asynctask  $(project)/app/asynctask/

withpgo: $(TARGETS)
	$(GO) build  -pgo=./sample.pgo -ldflags  '$(LDFLAGS)' -o grapes-app  $(project)/app/grapes/
	$(GO) build  -pgo=./sample.pgo -ldflags  '$(LDFLAGS)' -o grapes-vippay  $(project)/app/vippay/
	$(GO) build  -pgo=./sample.pgo -ldflags  '$(LDFLAGS)' -o grapes-llmchat  $(project)/app/llmchat/
	$(GO) build  -pgo=./sample.pgo -ldflags  '$(LDFLAGS)' -o grapes-asynctask  $(project)/app/asynctask/
	
image: $(TARGETS)
	tar cvf build.tar $(TARGETS)-app
	docker build -f dockerfiles/Dockerfile -t $(IMAGE) .
	rm -f build.tar 
	@echo "image: $(IMAGE)"

image-grapes:
	docker build -f dockerfiles/Dockerfile.grapes -t grapes-app:$(VERSION)-$(BUILD) .

image-vippay:
	docker build -f dockerfiles/Dockerfile.vippay -t grapes-vippay:$(VERSION)-$(BUILD) .

image-llmchat:
	docker build -f dockerfiles/Dockerfile.llmchat -t grapes-llmchat:$(VERSION)-$(BUILD) .

image-asynctask:
	docker build -f dockerfiles/Dockerfile.asynctask -t grapes-asynctask:$(VERSION)-$(BUILD) .

check:
	@$(GO) tool vet ${SRC}

test:
	@$(GO) test -race `$(GO) list ./... 

clean:
	rm -f $(TARGETS)-*

cov:
	gocov test -timeout=20m -race -v `$(GO) list ./... |egrep -v "app"`

cert:
	sh ./certs/gen.sh

build-all:
	$(MAKE) build-grapes
	$(MAKE) build-vippay
	$(MAKE) build-llmchat
	$(MAKE) build-asynctask
build-grapes:
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-app  $(project)/app/grapes/

build-vippay:
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-pay  $(project)/app/vippay/

build-llmchat:
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-llmchat  $(project)/app/llmchat/

build-asynctask:
	$(GO) build  -ldflags  '$(LDFLAGS)' -o grapes-asynctask  $(project)/app/asynctask/
