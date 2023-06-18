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

Note that if the command string uses `{}`,
the selected text is *not* passed via stdin.

## Execution context

The command string is executed directly by tmux-fastcopy,
so it must be a path to a binary or shell script that is executable.
It is not executed in the context of a full login shell.

The command runs inside the directory of the pane
where tmux-fastcopy was invoked if this information is available from tmux.
It runs with the following environment variables set:

- `FASTCOPY_REGEX_NAME`:
  Name of `@fastcopy-regex` rule that matched.
  See [Regex names](regex-names.md) and [Accessing the regex name](howto-regex-name.md)
  for more information.
- `FASTCOPY_TARGET_PANE_ID`:
  Unique identifier for the pane inside which fastcopy was invoked.
  Use this when running tmux operations inside the action
  to target them to that pane.
