# tmux-fastcopy [![Go](https://github.com/abhinav/tmux-fastcopy/actions/workflows/go.yml/badge.svg)](https://github.com/abhinav/tmux-fastcopy/actions/workflows/go.yml)

tmux-fastcopy aids in copying of text in a tmux pane with ease.

**How?** When you invoke tmux-fastcopy, it inspects your tmux pane and overlays
important pieces of text you may want to copy with very short labels that you
can use to copy them.

**Demos**: A gif is worth a paragraph or two.

<details>
  <summary>Git hashes</summary>

  ![git hashes demo](./static/gitlog.gif)
</details>

<details>
  <summary>File paths</summary>

  ![file paths demo](./static/files.gif)
</details>

<details>
  <summary>IP addresses</summary>

  ![IP addresses demo](./static/ip.gif)
</details>

<details>
  <summary>UUIDs</summary>

  ![UUIDs demo](./static/uuid.gif)
</details>

## Installation

The following methods of installation are available:

- via [Tmux Plugin Manager](#tmux-plugin-manager)
- [Manual installation](#manual-installation)
- [Binary installation](#binary-installation)

### Tmux Plugin Manager

**Prerequisite**: To use this method, you must have a Go compiler available on
your system.

If you're using [Tmux Plugin Manager](https://github.com/tmux-plugins/tpm), to
install, add tmux-fastcopy to the plugin list in your `.tmux.conf`:

    set -g @plugin 'abhinav/tmux-fastcopy'

Hit `<prefix> + I` to fetch and build it.

### Manual installation

**Prerequisite**: To use this method, you must have a Go compiler available on
your system.

Clone the repository somewhere on your system:

    git clone https://github.com/abhinav/tmux-fastcopy ~/.tmux/plugins/tmux-fastcopy

Source it in your `.tmux.conf`.

    run-shell ~/.tmux/plugins/tmux-fastcopy/fastcopy.tmux

Refresh your tmux server if it's already running.

    tmux source-file ~/.tmux.conf

### Binary installation

Alternatively, instead of instaling tmux-fastcopy as a tmux plugin, you can
install it as an independent binary.

1. Download a pre-built binary from the [releases page][] and place it on your
   `$PATH`.
2. Add the following to your `.tmux.conf`.

    ```
    bind-key f run-shell -b tmux-fastcopy
    ```

  [releases page]: https://github.com/abhinav/tmux-fastcopy/releases

## Usage

When there is text on the screen you'd like to copy:

1. Press `<prefix> + f` to invoke tmux-fastcopy. (You can change this key by
   setting the [`@fastcopy-key`][] option.
2. Enter the label next to the highlighted text to copy that text.

For example,

![IP addresses demo](./static/ip.gif)

By default, the copied text will be placed in your tmux buffer. Paste it by
pressing `<prefix> + ]`.

If you'd like to copy the text to your system clipboard, you can have
tmux-fastcopy do that by setting the [`@fastcopy-action`][] option. For
example, on macOS, add the following to your `~/.tmux.conf` to copy to the
system clipboard.

    set-option -g @fastcopy-action pbcopy

## Customization

Use the following tmux options to configure the behavior of tmux-fastcopy.

- [`@fastcopy-key`][] invokes tmux-fastcopy with the tmux prefix
- [`@fastcopy-action`][] copies the selected text
- [`@fastcopy-alphabet`][] specifies the letters used to generate labels

  [`@fastcopy-key`]: #fastcopy-key
  [`@fastcopy-action`]: #fastcopy-action
  [`@fastcopy-alphabet`]: #fastcopy-alphabet

### `@fastcopy-key`

Invoke tmux-fastcopy in tmux with this the `prefix` followed by this key.

**Default**:

    set-option -g @fastcopy-key f


### `@fastcopy-action`

Change how text is copied with this action.

**Default**:

    set-option -g @fastcopy-action 'tmux set-buffer -- {}'

The string specifies the command to run with the selection, as well as the
arguments for the command. The special argument `{}` acts as a placeholder for
the selected text.

If `{}` is absent from the command, tmux-fastcopy will pass the selected text
to the command over stdin. For example,

    set-option -g @fastcopy-action pbcopy  # for macOS

### `@fastcopy-alphabet`

Specify the letters used to generate labels for matched text.

**Default**:

    set-option -g @fastcopy-alphabet abcdefghijklmnopqrstuvwxyz

This must be a string containing at least two letters, and all of them must be
unique.

## Credits

The plugin is inspired by functionality provided by the [Vimium][] and
[Vimperator][] Chrome and Firefox plugins.

  [Vimium]: https://vimium.github.io/
  [Vimperator]: http://vimperator.org/vimperator

## Similar Projects

- [CrispyConductor/tmux-copy-toolkit](https://github.com/CrispyConductor/tmux-copy-toolkit)
- [fcsonline/tmux-thumbs](https://github.com/fcsonline/tmux-thumbs)
- [Morantron/tmux-fingers](https://github.com/Morantron/tmux-fingers)
