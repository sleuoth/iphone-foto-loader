#!/bin/bash
# Stub helper that returns fixture JSON for testing iphone-loader without a real iPhone.

set -e

SUBCMD="${1:-}"

case "$SUBCMD" in
  identify)
    echo '{"devices":[{"uuid":"stub-uuid-001","productName":"iPhone 15 Pro","deviceName":"Stub iPhone","isTrusted":true}]}'
    ;;
  list)
    echo '{"files":[
      {"handle":"h1","name":"IMG_1234.HEIC","size":1000,"created":"2026-06-27T10:30:00Z","mimeType":"image/heic","livePhotoPair":"IMG_1234.MOV"},
      {"handle":"h2","name":"IMG_1234.MOV","size":2000,"created":"2026-06-27T10:30:00Z","mimeType":"video/quicktime","livePhotoPair":"IMG_1234.HEIC"},
      {"handle":"h3","name":"IMG_1235.JPG","size":3000,"created":"2026-06-28T11:00:00Z","mimeType":"image/jpeg","livePhotoPair":null},
      {"handle":"h4","name":"whatsapp-image.jpg","size":500,"created":"2026-06-29T12:00:00Z","mimeType":"image/jpeg","livePhotoPair":null}
    ]}'
    ;;
  download)
    # parse --to argument
    TO=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --to) TO="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    if [ -z "$TO" ]; then
      echo "ERROR: missing --to" >&2
      exit 1
    fi
    echo "stub-file-content" > "$TO"
    ;;
  *)
    echo "ERROR: unknown subcommand: $SUBCMD" >&2
    exit 1
    ;;
esac
