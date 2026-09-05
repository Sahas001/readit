#!/usr/bin/env bash
set -euo pipefail

# tui-screen.sh
# Captures a tmux terminal pane/window and rasterizes it into a high-fidelity PNG screenshot.
# Usage: ./scripts/tui-screen.sh [TARGET_SESSION_OR_PANE] [OUTPUT_PNG_PATH] [WINDOW_SIZE]
# Example: ./scripts/tui-screen.sh dev /tmp/screen.png 1440,900

TARGET="${1:-dev}"
OUTPUT="${2:-/tmp/tui-screen.png}"
WINDOW_SIZE="${3:-1440,900}"

# Ensure required tools are available
for cmd in tmux aha google-chrome-stable; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: Required command '$cmd' is not installed or not in PATH." >&2
        exit 1
    fi
done

# Ensure parent directory of output exists
mkdir -p "$(dirname "$OUTPUT")"

TMP_HTML=$(mktemp /tmp/tui_capture_XXXXXX.html)
trap 'rm -f "$TMP_HTML"' EXIT

# Capture ANSI color terminal buffer from tmux
BODY=$(tmux capture-pane -e -p -t "$TARGET" | aha --no-header)

# Wrap in modern dark terminal canvas styling
cat << 'HEADER_EOF' > "$TMP_HTML"
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
body {
    background-color: #0b0b10;
    margin: 0;
    padding: 24px;
    display: flex;
    justify-content: center;
    align-items: flex-start;
}
pre {
    font-family: 'JetBrains Mono', 'Fira Code', 'DejaVu Sans Mono', 'Ubuntu Mono', monospace;
    font-size: 13.5px;
    line-height: 1.22;
    background-color: #12121a;
    color: #f0f0f5;
    padding: 18px 24px;
    border-radius: 10px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.85);
    border: 1px solid #2c2c3e;
    margin: 0;
    white-space: pre;
    tab-size: 4;
}
</style>
</head>
<body>
<pre>
HEADER_EOF

cat << BODY_EOF >> "$TMP_HTML"
${BODY}
</pre>
</body>
</html>
BODY_EOF

# Render headless screenshot
google-chrome-stable \
    --headless=new \
    --disable-gpu \
    --hide-scrollbars \
    --screenshot="$OUTPUT" \
    --window-size="$WINDOW_SIZE" \
    "$TMP_HTML" &>/dev/null

echo "TUI Screenshot successfully saved: $OUTPUT"
