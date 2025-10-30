#!/bin/bash
# Find macOS app executable path

if [ $# -eq 0 ]; then
    echo "Usage: $0 <app-name>"
    echo "Example: $0 Safari"
    exit 1
fi

APP_NAME="$1"

# Find app bundle
APP_BUNDLE=$(find /Applications -name "${APP_NAME}.app" -type d 2>/dev/null | head -1)

if [ -z "$APP_BUNDLE" ]; then
    echo "App '$APP_NAME' not found in /Applications"
    exit 1
fi

# Find executable inside bundle
EXECUTABLE=$(find "$APP_BUNDLE/Contents/MacOS" -type f -executable 2>/dev/null | head -1)

if [ -z "$EXECUTABLE" ]; then
    echo "No executable found in $APP_BUNDLE/Contents/MacOS"
    exit 1
fi

echo "App Bundle: $APP_BUNDLE"
echo "Executable: $EXECUTABLE"
echo ""
echo "To block this app, use:"
echo "keyphy add app $EXECUTABLE"
echo "or"
echo "keyphy add app $APP_NAME:$EXECUTABLE"