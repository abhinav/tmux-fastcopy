# Usage

When there is text on the screen you'd like to copy:

1. Press `<prefix> + f` to invoke tmux-fastcopy. (You can change this key by
   setting the [`@fastcopy-key`][] option.)
2. Enter the label next to the highlighted text to copy that text.

For example,

![IP addresses demo](./static/ip.gif)

By default, the copied text will be placed in your tmux buffer. Paste it by
pressing `<prefix> + ]`.

If you'd like to copy the text to your system clipboard, and you're using
tmux >= 3.2, add the following to your .tmux.conf:

    set-option -g set-clipboard on
    set-option -g @fastcopy-action 'tmux load-buffer -w -'

See [How to copy text to the clipboard?](howto-clipboard.md) for older versions of
tmux.
