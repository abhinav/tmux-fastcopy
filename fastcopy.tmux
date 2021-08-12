#!/usr/bin/env bash

FASTCOPY_KEY="$(tmux show-option -gqv @fastcopy-key)"
if [[ -z "$FASTCOPY_KEY" ]]; then
	FASTCOPY_KEY=f  # default
fi

FASTCOPY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FASTCOPY_EXE="$FASTCOPY_ROOT/bin/tmux-fastcopy"
if [[ ! -x "$FASTCOPY_EXE" ]]; then
	FASTCOPY_EXE=$(command -v tmux-fastcopy)
fi

if [[ -n "$FASTCOPY_EXE" ]]; then
	tmux bind-key "$FASTCOPY_KEY" run-shell -b "$FASTCOPY_EXE"
else
	if command -v go >/dev/null; then
		tmux display-message 'Building tmux-fastcopy...'
		tmux split-window -c "$FASTCOPY_ROOT" \
			-e "GOBIN=$FASTCOPY_ROOT/bin" \
			"go install github.com/abhinav/tmux-fastcopy"
	else
		tmux display-message -d 0 \
			"tmux-fastcopy not installed. Plese check the README."
	fi
fi
