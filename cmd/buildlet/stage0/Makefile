#
# The stage0 binaries are typically baked into VM images and container
# images and are NOT downloaded per build or VM/container start-up.
# The only reason we upload them to GCS is because that's where the
# automated VM/container creation scripts download them from.
#

GO=go1.19

ALL=\
	buildlet-stage0.illumos-amd64 \
	buildlet-stage0.linux-arm \
	buildlet-stage0.linux-arm64 \
	buildlet-stage0.linux-loong64 \
	buildlet-stage0.linux-mips \
	buildlet-stage0.linux-mips64 \
	buildlet-stage0.linux-mips64le \
	buildlet-stage0.linux-mipsle \
	buildlet-stage0.linux-ppc64 \
	buildlet-stage0.linux-ppc64le \
	buildlet-stage0.linux-s390x \
	buildlet-stage0.solaris-amd64 \
	buildlet-stage0.windows-amd64 \
	buildlet-stage0.windows-arm64 \

usage: FORCE
	@sed -E '/^$$/q; s/^# ?//' Makefile
	exit 1

FORCE:

# A convenience for people who are missing $(GO).
install-go:
	go install golang.org/dl/$(GO)@latest
	$(GO) download

clean:
	rm -f buildlet-stage0.*-*

# Compile everything without uploading.
compile: $(ALL)

GOFILES:=$(shell ls *.go)

BUILD=CGO_ENABLED=0 $(GO) build -o

buildlet-stage0.%-arm: $(GOFILES)
	GOOS=$* GOARCH=arm GOARM=7 $(BUILD) $@

buildlet-stage0.%-arm64: $(GOFILES)
	GOOS=$* GOARCH=arm64 $(BUILD) $@

buildlet-stage0.%-amd64: $(GOFILES)
	GOOS=$* GOARCH=amd64 $(BUILD) $@

buildlet-stage0.%-loong64: $(GOFILES)
	GOOS=$* GOARCH=loong64 $(BUILD) $@

buildlet-stage0.%-mips: $(GOFILES)
	GOOS=$* GOARCH=mips $(BUILD) $@

buildlet-stage0.%-mipsle: $(GOFILES)
	GOOS=$* GOARCH=mipsle $(BUILD) $@

buildlet-stage0.%-mips64: $(GOFILES)
	GOOS=$* GOARCH=mips64 $(BUILD) $@

buildlet-stage0.%-mips64le: $(GOFILES)
	GOOS=$* GOARCH=mips64le $(BUILD) $@

buildlet-stage0.%-ppc64: $(GOFILES)
	GOOS=$* GOARCH=ppc64 $(BUILD) $@

buildlet-stage0.%-ppc64le: $(GOFILES)
	GOOS=$* GOARCH=ppc64le $(BUILD) $@

buildlet-stage0.%-s390x: $(GOFILES)
	GOOS=$* GOARCH=s390x $(BUILD) $@

docker: Dockerfile
	go install golang.org/x/build/cmd/xb
	xb docker build --force-rm -f Dockerfile --tag=golang/buildlet-stage0 ../../..

# Upload everything.
upload: $(ALL:buildlet-stage0.%=upload.%)

UPLOAD=$(GO) run golang.org/x/build/cmd/upload -verbose -public -cacheable=false

upload.%: buildlet-stage0.%
	$(UPLOAD) -file=$< go-builder-data/$<

# Upload everything to dev cluster.
dev-upload: $(ALL:buildlet-stage0.%=upload.%)

dev-upload.%: buildlet-stage0.%
	$(UPLOAD) -file=$< dev-go-builder-data/$<
