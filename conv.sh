#!/bin/bash
# conv.sh - convert image(s) to webp, move result into <repo>/static/images
# usage: ./conv.sh photo.png [more.jpg ...]
# works no matter what directory you run it from

QUALITY=80

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST_DIR="$SCRIPT_DIR/static/images"

mkdir -p "$DEST_DIR"

if [ "$#" -eq 0 ]; then
    echo "usage: ./conv.sh image1.png [image2.jpg ...]"
    exit 1
fi

convert_file() {
    local input="$1"
    local out="$2"
    local ext="${input##*.}"

    case "$ext" in
        gif|GIF)
            gif2webp "$input" -o "$out"
            ;;
        *)
            cwebp -q "$QUALITY" "$input" -o "$out"
            ;;
    esac
}

for input in "$@"; do
    if [ ! -f "$input" ]; then
        echo "skip (not found): $input"
        continue
    fi

    filename="$(basename "$input")"
    name="${filename%.*}"
    out="$DEST_DIR/$name.webp"

    if [ -f "$out" ]; then
        echo "conflict: $out already exists."
        read -p "  (o)verwrite / (r)ename / (s)kip? " choice
        case "$choice" in
            o|O)
                convert_file "$input" "$out"
                echo "-> overwritten: $out"
                ;;
            r|R)
                read -p "  new name (without extension): " newname
                out="$DEST_DIR/$newname.webp"
                convert_file "$input" "$out"
                echo "-> $out"
                ;;
            *)
                echo "skipped: $input"
                ;;
        esac
    else
        convert_file "$input" "$out"
        echo "-> $out"
    fi
done
