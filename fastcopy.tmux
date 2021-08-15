#!/usr/bin/env bash

FASTCOPY_KEY="$(tmux show-option -gqv @fastcopy-key)"
if [[ -z "$FASTCOPY_KEY" ]]; then
	FASTCOPY_KEY=f  # default
fi

FASTCOPY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FASTCOPY_EXE="$FASTCOPY_ROOT/bin/tmux-fastcopy"
if [ ! -x "$FASTCOPY_EXE" ]; then
	if command -v tmux-fastcopy >/dev/null; then
		# Fall back to a globally installed version if available.
		FASTCOPY_EXE=tmux-fastcopy
	else
		tmux display-message 'Installing tmux-fastcopy locally...'
		tmux split-window -c "$FASTCOPY_ROOT" "$FASTCOPY_ROOT/install.sh"
	fi
fi

tmux bind-key "$FASTCOPY_KEY" run-shell -b "$FASTCOPY_EXE"
