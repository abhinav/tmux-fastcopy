# Copy text to the clipboard?

To copy text to your system clipboard, you can use tmux's `set-clipboard`
option and change the action to `tmux load-buffer -w -` if you're using
at least tmux 3.2.

    set-option -g set-clipboard on
    set-option -g @fastcopy-action 'tmux load-buffer -w -'

With this option set, and the `-w` flag for `load-buffer`, tmux will use the
OSC52 escape sequence to directly set the clipboard for your terminal
emulator--it should work even through an SSH session. Check out
[A guide on how to copy text from anywhere][osc52] to read more about OSC52.

  [osc52]: https://old.reddit.com/r/vim/comments/k1ydpn/a_guide_on_how_to_copy_text_from_anywhere/

If you're using an older version of tmux or your terminal emulator does not
support OSC52,  you can configure `@fastcopy-action` to have tmux-fastcopy
send the text elsewhere. For example,

    # On macOS:
    set-option -g @fastcopy-action pbcopy

    # For Linux systems using X11, install [xclip] and use:
    #
    #  [xclip]: https://github.com/astrand/xclip
    set-option -g @fastcopy-action 'xclip -selection clipboard'

    # For Linux systems using Wayland, install [wl-clipboard] and use:
    #
    #  [wl-clipboard]: https://github.com/bugaevc/wl-clipboard
    set-option -g @fastcopy-action wl-copy
