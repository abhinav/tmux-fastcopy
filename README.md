# tmux-fastcopy [![Go](https://github.com/abhinav/tmux-fastcopy/actions/workflows/go.yml/badge.svg)](https://github.com/abhinav/tmux-fastcopy/actions/workflows/go.yml)

tmux-fastcopy aids in copying of text in a tmux pane with ease by providing
vimium/vimperator-style type hints.

## Configuration

Use the following tmux options to configure the behavior of tmux-fastcopy.

- [`@fastcopy-key`][] invokes tmux-fastcopy with the tmux prefix
- [`@fastcopy-action`][] copies the selected text
- [`@fastcopy-alphabet`][] specifies the letters used to generate labels

  [`@fastcopy-key`]: #fastcopy-key
  [`@fastcopy-action`]: #fastcopy-action
  [`@fastcopy-alphabet`]: #fastcopy-alphabet

### `@fastcopy-key`

Invoke tmux-fastcopy in tmux with this the `prefix` followed by this key.
Default:

```
set-option -g @fastcopy-key f
```


### `@fastcopy-action`

Change how text is copied with this action.
Default:

```
set-option -g @fastcopy-action 'tmux set-buffer -- {}'
```

The string specifies the command to run with the selection, as well as the
arguments for the command. The special argument `{}` acts as a placeholder for
the selected text.

If `{}` is absent from the command, tmux-fastcopy will pass the selected text
to the command over stdin. For example,

```
set-option -g @fastcopy-action pbcopy  # for macOS
```

### `@fastcopy-alphabet`

Specify the letters used to generate labels for matched text.
Default:

```
set-option -g @fastcopy-alphabet abcdefghijklmnopqrstuvwxyz
```

This must be a string containing at least two letters, and all of them must be
unique.

## Similar projects

- [CrispyConductor/tmux-copy-toolkit](https://github.com/CrispyConductor/tmux-copy-toolkit)
- [fcsonline/tmux-thumbs](https://github.com/fcsonline/tmux-thumbs)
- [Morantron/tmux-fingers](https://github.com/Morantron/tmux-fingers)
