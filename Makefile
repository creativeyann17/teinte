.PHONY: install build build-linux dev run test icon

# Regenerate the app icon (PNG + Windows ICO). The ImageMagick step
# matters: small ICO entries must be BMP-encoded or Windows Explorer
# may show a blank icon (only the 256px entry may be PNG).
icon:
	python3 build/make_icon.py
	convert build/appicon.png -define icon:auto-resize=256,128,64,48,32,16 build/windows/icon.ico

install:
	go mod download
	cd frontend && bun install

# Windows is the real target (gamma ramp + NvAPI); pure-Go, cross-compiles from Linux.
build:
	wails build -platform windows/amd64

# Ubuntu ships webkit2gtk-4.1, hence the tag.
build-linux:
	wails build -platform linux/amd64 -tags webkit2_41

dev:
	wails dev -tags webkit2_41

run: build-linux
	./build/bin/teinte

test:
	go test ./...
