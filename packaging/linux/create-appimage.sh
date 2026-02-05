#!/bin/bash
set -e

# Usage: ./create-appimage.sh <version> <arch> <source_dir> <output_dir>
VERSION="$1"
ARCH="$2"
SOURCE_DIR="$3"
OUTPUT_DIR="$4"

if [ -z "$VERSION" ] || [ -z "$ARCH" ] || [ -z "$SOURCE_DIR" ] || [ -z "$OUTPUT_DIR" ]; then
    echo "Usage: $0 <version> <arch> <source_dir> <output_dir>"
    exit 1
fi

echo "Building AppImage for bish v${VERSION} (${ARCH})"

# Create AppDir structure
APPDIR="bish.AppDir"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
mkdir -p "$APPDIR/usr/share/applications"
mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$APPDIR/usr/share/doc/bish"
mkdir -p "$APPDIR/usr/share/licenses/bish"

# Copy binary
cp "$SOURCE_DIR/bish" "$APPDIR/usr/bin/"
chmod +x "$APPDIR/usr/bin/bish"

# Copy documentation
cp "$SOURCE_DIR/LICENSE" "$APPDIR/usr/share/licenses/bish/"
cp "$SOURCE_DIR/README.md" "$APPDIR/usr/share/doc/bish/"

# Create desktop file
cat > "$APPDIR/bish.desktop" << 'EOF'
[Desktop Entry]
Type=Application
Name=bish
Comment=A modern, POSIX-compatible, generative shell
Exec=bish
Icon=bish
Categories=System;TerminalEmulator;
Terminal=true
EOF

# Copy desktop file to standard location
cp "$APPDIR/bish.desktop" "$APPDIR/usr/share/applications/"

# Copy icon from assets (use existing project icon)
if [ -f "assets/images/icon.png" ]; then
    cp "assets/images/icon.png" "$APPDIR/bish.png"
    cp "$APPDIR/bish.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/"
elif [ -f "docs/images/icon.png" ]; then
    cp "docs/images/icon.png" "$APPDIR/bish.png"
    cp "$APPDIR/bish.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/"
else
    # Fallback: create a simple terminal icon if no icon exists
    cat > "$APPDIR/bish.svg" << 'EOF'
<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256">
  <rect width="256" height="256" fill="#1a1a1a" rx="20"/>
  <text x="128" y="160" font-family="monospace" font-size="120" fill="#00ff00" text-anchor="middle">$</text>
</svg>
EOF
    # Use SVG if no PNG icon available
    cp "$APPDIR/bish.svg" "$APPDIR/usr/share/icons/hicolor/256x256/apps/bish.svg"
fi

# Create AppRun launcher
cat > "$APPDIR/AppRun" << 'EOF'
#!/bin/bash
SELF=$(readlink -f "$0")
HERE=${SELF%/*}
export PATH="${HERE}/usr/bin:${PATH}"
exec "${HERE}/usr/bin/bish" "$@"
EOF
chmod +x "$APPDIR/AppRun"

# Download appimagetool for the target architecture
APPIMAGETOOL="appimagetool-${ARCH}.AppImage"
if [ ! -f "$APPIMAGETOOL" ]; then
    echo "Downloading appimagetool for ${ARCH}..."
    curl -L -o "$APPIMAGETOOL" \
        "https://github.com/AppImage/appimagetool/releases/download/continuous/${APPIMAGETOOL}"
    chmod +x "$APPIMAGETOOL"
fi

# Build AppImage
export ARCH="$ARCH"
./"$APPIMAGETOOL" "$APPDIR" "$OUTPUT_DIR/bish-${VERSION}-${ARCH}.AppImage"

echo "AppImage created: $OUTPUT_DIR/bish-${VERSION}-${ARCH}.AppImage"
