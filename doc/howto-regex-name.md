# Access the regex name

tmux-fastcopy executes the action with the `FASTCOPY_REGEX_NAME` environment
variable set. This holds the [name of the regex](regex-names.md) that matched the
selected string.
If multiple different regexes matched the string, `FASTCOPY_REGEX_NAME` holds a
space-separated list of them.

You can use this to customize the action on a per-regex basis.

For example, the following will copy most strings to the tmux buffer as usual.
However, if the string is matched by the "path" regular expression and it
represents an existing directory, this will open that directory in the file
browser.

```bash
#!/usr/bin/env bash

# Place this inside a file like "fastcopy.sh",
# mark it executable (chmod +x fastcopy.sh),
# and set the @fastcopy-action setting to:
#   '/path/to/fastcopy.sh {}'

if [ "$FASTCOPY_REGEX_NAME" == path ] && [ -d "$1" ]; then
    xdg-open "$1"  # on macOS, use "open" instead
    exit 0
fi

tmux set-buffer -w "$1"
```
