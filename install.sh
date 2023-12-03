#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Places a copy of tmux-fastcopy at bin/tmux-fastcopy.

# Invoke with "-c $VERSION" to check if this would install "$VERSION".

IMPORTPATH=github.com/abhinav/tmux-fastcopy
NAME=tmux-fastcopy
VERSION=0.14.1

while getopts 'c:' opt; do
	case "$opt" in
		c)
			if [[ "$VERSION" == "$OPTARG" ]]; then
				echo >&2 "Versions match!"
				exit 0
			fi
			echo >&2 "Version mismatch:"
			echo >&2 "  want: $VERSION"
			echo >&2 "   got: $OPTARG"
			exit 1
			;;
		'?')
			exit 1
	esac
done
shift "$((OPTIND-1))"

OS=$(uname -s)
ARCH=$(uname -m)

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINDIR="$PROJECT_ROOT/bin"
EXE="$BINDIR/$NAME"

DOWNLOADS_URL="https://$IMPORTPATH/releases/download"

# Build from source if go is available.
# No arguments.
try_build() {
	command -v go >/dev/null &&
		echo >&2 "Building from source..." &&
		(cd "$PROJECT_ROOT" && go build -o "$EXE")
}

# Downoad with curl. Takes the URL as an argument.
try_curl() {
	command -v curl >/dev/null &&
		echo >&2 "Downloading with curl..." &&
		curl -L -o >(tar -xvz -C "$BINDIR" "$NAME") "$1"
}

# Download with wget. Takes the URL as an argument.
try_wget() {
	command -v wget >/dev/null &&
		echo >&2 "Downloading with wget..." &&
		wget -O >(tar -xvz -C "$BINDIR" "$NAME") "$1"
}

# Downloads a pre-built binary.
try_download() {
	tarball="${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
	url="$DOWNLOADS_URL/v${VERSION}/$tarball"

	mkdir -p "$BINDIR"
	if (try_curl "$url") || (try_wget "$url"); then
		chmod +x "$EXE"
	fi
}

if ! (try_build || try_download); then
	echo >&2 "Unable to build or download $NAME."
	echo >&2 "This means,"
	echo >&2 "  1. You do not have Go installed; and"
	echo >&2 "  2. You are using an OS, architecutre, or version for which we"
	echo >&2 "     do not distribute pre-built binaries."
	echo >&2 "Please resolve one of these issues and try again."
	echo >&2
	echo >&2 "Press any key to continue:"
	read -rk1
	exit 1
fi
