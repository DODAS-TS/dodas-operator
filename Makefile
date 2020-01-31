VERSION?=`git describe --tags`
DOCBIN?=mkdocs

.PHONY: init build push publish-doc

all: init build

help:
	@echo "Available commands:\n"
	@echo "- build			: "
	@echo "- init			: "
	@echo "- push			: "
	@echo "- publish-doc			: "

init:
	.scripts/install-sdk.sh

build:
	./scripts/build.sh

push:
	./scripts/push.sh

publish-doc:
	cp README.md docs/README.md
	$(DOCBIN) gh-deploy

