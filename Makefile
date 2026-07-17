.PHONY: install build build-linux dev run test icon appimage

# Overridable, e.g. CI: make appimage LDFLAGS="-s -w -X main.version=1.2.3"
LDFLAGS ?=

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
	wails build -platform linux/amd64 -tags webkit2_41 -ldflags "$(LDFLAGS)"

dev:
	wails dev -tags webkit2_41

run: build-linux
	./build/bin/teinte

test:
	go test ./...

# Packages build/bin/teinte as an AppImage (same tool/flags as CI's
# release.yml). linuxdeploy + its plugin are cached in build/bin/
# (gitignored) so repeat runs skip the download.
appimage: build-linux
	@test -x build/bin/linuxdeploy.AppImage || { \
		curl -Lo build/bin/linuxdeploy.AppImage https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage; \
		chmod +x build/bin/linuxdeploy.AppImage; \
	}
	@test -x build/bin/linuxdeploy-plugin-appimage.AppImage || { \
		curl -Lo build/bin/linuxdeploy-plugin-appimage.AppImage https://github.com/linuxdeploy/linuxdeploy-plugin-appimage/releases/download/continuous/linuxdeploy-plugin-appimage-x86_64.AppImage; \
		chmod +x build/bin/linuxdeploy-plugin-appimage.AppImage; \
	}
	rm -rf build/bin/appdir
	APPIMAGE_EXTRACT_AND_RUN=1 PATH="$(CURDIR)/build/bin:$$PATH" build/bin/linuxdeploy.AppImage \
		--appdir build/bin/appdir \
		--executable build/bin/teinte \
		--desktop-file build/linux/teinte.desktop \
		--icon-file build/linux/teinte.png \
		--output appimage
	mv Teinte*.AppImage build/bin/teinte-x86_64.AppImage
	@echo "AppImage: build/bin/teinte-x86_64.AppImage"
