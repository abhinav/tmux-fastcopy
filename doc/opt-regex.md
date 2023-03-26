# `@fastcopy-regex-*`

These specify the regular expressions used to match text.

**Default**:

	set-option -g @fastcopy-regex-ipv4 "\\b\\d{1,3}(?:\\.\\d{1,3}){3}\\b"
	set-option -g @fastcopy-regex-gitsha "\\b[0-9a-f]{7,40}\\b"
	set-option -g @fastcopy-regex-hexaddr "\\b(?i)0x[0-9a-f]{2,}\\b"
	set-option -g @fastcopy-regex-hexcolor "(?i)#(?:[0-9a-f]{3}|[0-9a-f]{6})\\b"
	set-option -g @fastcopy-regex-int "(?:-?|\\b)\\d{4,}\\b"
	set-option -g @fastcopy-regex-path "(?:[\\w\\-\\.]+|~)?(?:/[\\w\\-\\.]+){2,}\\b"
	set-option -g @fastcopy-regex-uuid "\\b(?i)[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}\\b"
	set-option -g @fastcopy-regex-isodate "\\d{4}-\\d{2}-\\d{2}"

Add new regular expressions by introducing new options with the prefix,
`@fastcopy-regex-`. For example, the following will match Phabricator revision
IDs if they're at least three letters long.

    set-option -g @fastcopy-regex-phab-diff "\\bD\\d{3,}\\b"

**Note**: You must double all `\` symbols inside regular expressions to
escape them properly.

<aside>

  > Read [this FAQ entry](faq.md#word-boundary) for an explanation of the `\\b`s
  > inside the regular expressions above.

</aside>

## Copying substrings

Use regex capturing groups if you wish to copy only a portion of the matched
string. tmux-fastcopy will copy the contents of the first capturing group. For
example,

    set-option -g @fastcopy-regex-python-import "import ([\\w\\.]+)"
    # From "import os.path", copy only "os.path"

This also means that to use `(...)` in regular expressions that should copy the
whole string, you should add the `?:` prefix to the start of the capturing
group to ignore it. For example,

    # Matches commands suggested by 'git status'
    set-option -g @fastcopy-regex-git-rebase "git rebase --(?:continue|abort)"
