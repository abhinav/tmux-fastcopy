# FAQ

## <a id="word-boundary"></a> What's the `\b` at the ends of some regexes?

The `\b` at either end of the regular expression above specifies that it must
start and/or end at a word boundary. A word boundary is the start or end of a
line, or a non-alphanumeric character.

For example, the regular expression `\bgit\b` will match the string `git`
inside `git rebase --continue` and `git-rebase`, but not inside `github`
because the "h" following the "git" is not a word boundary.

## The entire string did not get copied

If your regular expression uses capturing groups `(...)`, tmux-fastcopy will
only copy the first of these from the matched string.

In the regex below, only the strings "continue" or "abort" will be copied.

    set-option -g @fastcopy-regex-git-rebase "git rebase --(continue|abort)"

To copy the entire string, you can put the whole string in a capturing group,
making it the first capturing group.

    set-option -g @fastcopy-regex-git-rebase "(git rebase --(continue|abort))"

Or you can mark the `(continue|abort)` group as ignored by starting it with
`?:`.

    set-option -g @fastcopy-regex-git-rebase "git rebase --(?:continue|abort)"

## Are regular expressions case sensitive?

Yes, the regular expressions matched by tmux-fastcopy are case sensitive. For
example,

    set-option -g @fastcopy-regex-github-project "github.com/(\w+/\w+)"

This will match `github.com/abhinav/tmux-fastcopy` but not
`GitHub.com/abhinav/tux-fastcopy`.

If you want to turn your regular expression case insensitive, prefix it with
`(?i)`.

    set-option -g @fastcopy-regex-github-project "(?i)github.com/(\w+/\w+)"

## How to overwrite or remove default regexes?

To overwrite or remove default regular expressions, add a new regex to your
`tmux.conf` with the same name as the default one, using a blank string as the
value to delete it.

For example, the following deletes the `isodate` regular expression.

    set-option -g @fastcopy-regex-isodate ""

## Can I have different actions for different regexes?

The `FASTCOPY_REGEX_NAME` environment variable holds the name of the regex that
matched your selection.
You can run different actions on a per-regex basis by inspecting the
`FASTCOPY_REGEX_NAME` environment variable in your
[`@fastcopy-action`](opt-action.md).

See [Accessing the regex name](howto-regex-name.md) for more details.
