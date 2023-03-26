# `@fastcopy-action`

Change how text is copied with this action.

**Default**:

    set-option -g @fastcopy-action 'tmux load-buffer -'

The string specifies the command to run with the selection, as well as the
arguments for the command. The special argument `{}` acts as a placeholder for
the selected text.

    set-option -g @fastcopy-action 'tmux set-buffer {}'

If `{}` is absent from the command, tmux-fastcopy will pass the selected text
to the command over stdin. For example,

    set-option -g @fastcopy-action pbcopy  # for macOS

