# Select text without copying

If you'd like to select the matched text rather than copy in,
you can define an action that takes the target pane in copy mode,
and moves your cursor over to the matched text.

The following script should suffice for this:

```bash
#!/usr/bin/env bash

MATCH_TEXT="$1"
PANE_ID="$FASTCOPY_TARGET_PANE_ID"

tmux \
	copy-mode -t "$PANE_ID" ';' \
	send-keys -t "$PANE_ID" -X search-backward-text "$MATCH_TEXT" ';' \
	send-keys -t "$PANE_ID" -X begin-selection ';' \
	send-keys -t "$PANE_ID" -X -N "$((${#MATCH_TEXT} - 1))" cursor-right ';' \
	send-keys -t "$PANE_ID" -X end-selection
```

<details>
<summary>Explanation</summary>

The script above expects the matched text as an argument,
and grabs the target pane ID from the environment.
tmux-fastcopy sets `FASTCOPY_TARGET_PANE_ID` when running the action
(see [Execution context](opt-action.md#execution-context)).

It then runs the following tmux commands on the pane:

- switch it to copy mode
- search for the closest recent instance of the matched text
  and move your cursor there
- begin a selection
- move the cursor to the end of the selected text
- end the selection

The end result of this is that when the action runs,
your cursor will have selected the matched text
leaving you room to adjust the selection before copying.

</details>

Place this script in a location of your choice, say, `~/.tmux/select.sh`
and mark it as an executable:

```bash
chmod +x ~/.tmux/select.sh
```

Then add the following to your `~/tmux.conf`.

```tmux
set -g @fastcopy-action "~/.tmux/select.sh {}"
```

Or add the following if you want to do this
only when you press shift along with the label
(see [`@fastcopy-shift-action`](opt-shift-action.md)).

```tmux
set -g @fastcopy-shift-action "~/.tmux/select.sh {}"
```
