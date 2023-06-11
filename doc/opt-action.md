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

Note that the command string is executed through the tmux-fastcopy binary,
so it must be a path to a binary or shell script that is executable,
and is not executed in the context of a full login shell.
Additionally, if the command string uses `{}`,
the selected text is *not* passed via stdin.
