# Installs into ~/.local by default; override with e.g. `make install PREFIX=/usr/local`.
PREFIX ?= $(HOME)/.local
ICON_SIZES := 16 22 24 32 48 64 128 256 512
DESKTOP_FILE := $(PREFIX)/share/applications/translator.desktop

.PHONY: build install uninstall test vet

# The wails.json preBuildHooks entry renders build/icons/*.png and
# build/appicon.png before compiling, so `wails build` is self-contained.
# The webkit2_41 tag is required on systems with webkit2gtk-4.1 (the
# documented dependency); drop it only on older webkit2gtk-4.0 distros.
build:
	wails build -tags webkit2_41

install: build
	install -Dm755 build/bin/translator $(PREFIX)/bin/translator
	for size in $(ICON_SIZES); do \
		install -Dm644 build/icons/$$size.png \
			$(PREFIX)/share/icons/hicolor/$${size}x$${size}/apps/translator.png; \
	done
	mkdir -p $(dir $(DESKTOP_FILE))
	printf '%s\n' \
		'[Desktop Entry]' \
		'Type=Application' \
		'Name=LibreTranslate Translator' \
		'Comment=Desktop client for LibreTranslate' \
		'Exec=$(PREFIX)/bin/translator' \
		'Icon=translator' \
		'Terminal=false' \
		'Categories=Utility;Office;' \
		'StartupWMClass=translator' \
		> $(DESKTOP_FILE)
	@# refresh the icon cache only where one already exists (a stale cache hides icons)
	-test -f $(PREFIX)/share/icons/hicolor/icon-theme.cache && \
		gtk-update-icon-cache --force --ignore-theme-index $(PREFIX)/share/icons/hicolor
	@echo "Installed to $(PREFIX)/bin/translator"

uninstall:
	rm -f $(PREFIX)/bin/translator
	rm -f $(DESKTOP_FILE)
	for size in $(ICON_SIZES); do \
		rm -f $(PREFIX)/share/icons/hicolor/$${size}x$${size}/apps/translator.png; \
	done
	@echo "Uninstalled from $(PREFIX)"

test:
	go vet ./...
	go test -race ./...

vet:
	go vet ./...
