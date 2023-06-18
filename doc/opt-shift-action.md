# `@fastcopy-shift-action`

An alternative action when you select a label while pressing shift.
Nothing happens if this is unset.

**Default**:

    set-option -g @fastcopy-shift-action ''

Similarly to [`@fastcopy-action`], the string specifies a command and its
arguments, and the special argument `{}` (if any) is a placeholder for the
selected text.

    set-option -g @fastcopy-shift-action "fastcopy-shift.sh {}"

The `@fastcopy-shift-action` will run with the same
[execution context](opt-action.md#execution-context)
as the `@fastcopy-action`.
