# `@fastcopy-alphabet`

Specify the letters used to generate labels for matched text.

**Default**:

    set-option -g @fastcopy-alphabet abcdefghijklmnopqrstuvwxyz

This must be a string containing at least two letters, and all of them must be
unique.

For example, if you want to only use the letters from the QWERTY home row, use
the following.

    set-option -g @fastcopy-alphabet asdfghjkl
