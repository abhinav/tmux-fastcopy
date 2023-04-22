# Regex names

The portion after the `@fastcopy-regex-` can be any name that uniquely
identifies this regular expression.

For example, the name of this regular expression is `phab-diff`

    set-option -g @fastcopy-regex-phab-diff "\\bD\\d{3,}\\b"

You cannot have multiple regular expressions with the same name. New regular
expressions with previously used names will overwrite them. For example, this
overwrites the default `hexcolor` regular expression to copy only the color
code, skipping the preceding `#`:

	set-option -g @fastcopy-regex-hexcolor "(?i)#([0-9a-f]{3}|[0-9a-f]{6})\\b"

You can delete previously defined or default regular expressions by setting
them to a blank string.

    set-option -g @fastcopy-regex-isodate ""

The name of the regular expression that matched the selection is available to
the [`@fastcopy-action`](opt-action.md) via the `FASTCOPY_REGEX_NAME` environment variable.
See [Accessing the regex name](howto-regex-name.md) for more details.
