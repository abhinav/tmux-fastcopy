# `@fastcopy-shift-action`

An alternative action when you select a label while pressing shift.
Nothing happens if this is unset.

**Default**:

    set-option -g @fastcopy-shift-action ''

Similarly to [`@fastcopy-action`], the string specifies a command and its
arguments, and the special argument `{}` (if any) is a placeholder for the
selected text.

    set-option -g @fastcopy-shift-action "fastcopy-shift.sh {}"

As with `@fastcopy-action`, tmux-fastcopy will set `FASTCOPY_REGEX_NAME` to the
name of the regular expression that matched when running the
`@fastcopy-shift-action`.
See [Accessing the regex name](howto-regex-name.md) for more details.
