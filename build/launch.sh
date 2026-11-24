#!/bin/sh
PAK_DIR="$(cd "$(dirname "$0")" && pwd)"
PAK_NAME="$(basename "$PAK_DIR")"
PAK_NAME="${PAK_NAME%.*}"

mkdir -p "$LOGS_PATH"
exec >>"$LOGS_PATH/$PAK_NAME.txt" 2>&1

exec "$PAK_DIR/pocket-tunnel" menu \
    --state-dir "$USERDATA_PATH/$PAK_NAME" \
    --bin-dir "$PAK_DIR/bin/$PLATFORM" \
    --auto-sh "$USERDATA_PATH/auto.sh" \
    --log-file "$LOGS_PATH/$PAK_NAME-tunnel.txt"
