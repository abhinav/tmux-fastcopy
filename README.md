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

Alternatively, instead of installing tmux-fastcopy as a tmux plugin, you can
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
   setting the [`@fastcopy-key`][] option.)
2. Enter the label next to the highlighted text to copy that text.

For example,

![IP addresses demo](./static/ip.gif)

By default, the copied text will be placed in your tmux buffer. Paste it by
pressing `<prefix> + ]`.

If you'd like to copy the text to your system clipboard, use the tmux
`set-clipboard` option. See [How do I copy text to my clipboard?](#clipboard)
for more information.

    set-option -g set-clipboard on

## Customization

Use the following tmux options to configure the behavior of tmux-fastcopy.

- [`@fastcopy-key`][] invokes tmux-fastcopy with the tmux prefix
- [`@fastcopy-action`][] copies the selected text
- [`@fastcopy-alphabet`][] specifies the letters used to generate labels
- [`@fastcopy-regex-*`][] specify the regular expressions for matching text

  [`@fastcopy-key`]: #fastcopy-key
  [`@fastcopy-action`]: #fastcopy-action
  [`@fastcopy-alphabet`]: #fastcopy-alphabet
  [`@fastcopy-regex-*`]: #fastcopy-regex-

### `@fastcopy-key`

Invoke tmux-fastcopy in tmux with this the `prefix` followed by this key.

**Default**:

    set-option -g @fastcopy-key f


### `@fastcopy-action`

Change how text is copied with this action.

**Default**:

    set-option -g @fastcopy-action 'tmux set-buffer -w -- {}'

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

For example, if you want to only use the letters from the QWERTY home row, use
the following.

    set-option -g @fastcopy-alphabet asdfghjkl

### `@fastcopy-regex-*`

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

  > Read [this FAQ entry](#word-boundary) for an explanation of the `\\b`s
  > inside the regular expressions above.

</aside>

#### Copying substrings

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

#### Regex names

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

## FAQ

### <a id="clipboard"></a> How do I copy text to my clipboard?

To copy text to your system clipboard, you can use tmux's `set-clipboard`
option.

    set-option -g set-clipboard on

With this option set, tmux will make use of the OSC52 escape sequence to
directly set the clipboard for your terminal emulator--it should work even
through an SSH session. Also check out [A guide on how to copy text from anywhere][osc52]
to read more about OSC52.

  [osc52]: https://old.reddit.com/r/vim/comments/k1ydpn/a_guide_on_how_to_copy_text_from_anywhere/

If your terminal emulator does not support OSC52, you can configure
`@fastcopy-action` to have tmux-fastcopy send the text elsewhere. For example,

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

### <a id="word-boundary"></a> What's the `\b` at the ends of some regexes?

The `\b` at either end of the regular expression above specifies that it must
start and/or end at a word boundary. A word boundary is the start or end of a
line, or a non-alphanumeric character.

For example, the regular expression `\bgit\b` will match the string `git`
inside `git rebase --continue` and `git-rebase`, but not inside `github`
because the "h" following the "git" is not a word boundary.

### The entire string did not get copied

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

### Are regular expressions case sensitive?

Yes, the regular expressions matched by tmux-fastcopy are case sensitive. For
example,

    set-option -g @fastcopy-regex-github-project "github.com/(\w+/\w+)"

This will match `github.com/abhinav/tmux-fastcopy` but not
`GitHub.com/abhinav/tux-fastcopy`.

If you want to turn your regular expression case insensitive, prefix it with
`(?i)`.

    set-option -g @fastcopy-regex-github-project "(?i)github.com/(\w+/\w+)"

### How to overwrite or remove default regexes?

To overwrite or remove default regular expressions, add a new regex to your
`tmux.conf` with the same name as the default one, using a blank string as the
value to delete it.

For example, the following deletes the `isodate` regular expression.

    set-option -g @fastcopy-regex-isodate ""

## Credits

The plugin is inspired by functionality provided by the [Vimium][] and
[Vimperator][] Chrome and Firefox plugins.

  [Vimium]: https://vimium.github.io/
  [Vimperator]: http://vimperator.org/vimperator

## Similar Projects

- [CrispyConductor/tmux-copy-toolkit](https://github.com/CrispyConductor/tmux-copy-toolkit)
- [fcsonline/tmux-thumbs](https://github.com/fcsonline/tmux-thumbs)
- [Morantron/tmux-fingers](https://github.com/Morantron/tmux-fingers)
