# Installation

Before you install, make sure you are running a supported version of tmux.

```
$ tmux -V
```

Minimum supported version: 2.7.

The following methods of installation are available:

- via [Tmux Plugin Manager](#tmux-plugin-manager)
- [Manual installation](#manual-installation)
- [Binary installation](#binary-installation)

## Tmux Plugin Manager

**Prerequisite**: To use this method, you must have a Go compiler available on
your system.

If you're using [Tmux Plugin Manager](https://github.com/tmux-plugins/tpm), to
install, add tmux-fastcopy to the plugin list in your `.tmux.conf`:

    set -g @plugin 'abhinav/tmux-fastcopy'

Hit `<prefix> + I` to fetch and build it.

## Manual installation

**Prerequisite**: To use this method, you must have a Go compiler available on
your system.

Clone the repository somewhere on your system:

    git clone https://github.com/abhinav/tmux-fastcopy ~/.tmux/plugins/tmux-fastcopy

Source it in your `.tmux.conf`.

    run-shell ~/.tmux/plugins/tmux-fastcopy/fastcopy.tmux

Refresh your tmux server if it's already running.

    tmux source-file ~/.tmux.conf

## Binary installation

Instead of installing tmux-fastcopy as a tmux plugin,
you can install it as an independent binary.

Use one of the following to install the binary.

- If you're using **Homebrew**/Linuxbrew, run:

    ```bash
    brew install abhinav/tap/tmux-fastcopy
    ```

- If you're using **ArchLinux**, install it from AUR using the [tmux-fastcopy](https://aur.archlinux.org/packages/tmux-fastcopy/) package,
  or the [tmux-fastcopy-bin](https://aur.archlinux.org/packages/tmux-fastcopy-bin/) package if you don't want to build it from source.

    ```bash
    git clone https://aur.archlinux.org/tmux-fastcopy.git
    cd tmux-fastcopy
    makepkg -si
    ```

  With an AUR helper like [yay](https://github.com/Jguer/yay), run:

    ```bash
    yay -S tmux-fastcopy
    # or
    yay -S tmux-fastcopy-bin
    ```

- Download a **pre-built binary** from the [releases page](https://github.com/abhinav/tmux-fastcopy/releases)
  and place it on your `$PATH`.

- Build it from source with Go.

    ```bash
    go install github.com/abhinav/tmux-fastcopy@latest
    ```

Once you have the binary installed, add the following to your `.tmux.conf`.

```
bind-key f run-shell -b tmux-fastcopy
```
