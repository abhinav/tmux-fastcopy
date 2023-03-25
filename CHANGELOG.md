# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.10.0 - 2023-03-25
### Changed
- Use event-based rendering instead of fixed rate rendering.
  This should reduce flickering on slow systems.

## 0.9.0 - 2022-05-29
### Added
- Add `@fastcopy-shift-action` to specify an alternative action to be run when
  a label is selected with the Shift key pressed.

## 0.8.0 - 2022-04-01
### Added
- Expose the name of the matched regex to the action with the
  `FASTCOPY_REGEX_NAME` environment variable.

## 0.7.2 - 2022-02-19
### Added
- For 32-bit ARM binaries, support ARM v5, v6, and v7.

## 0.7.1 - 2022-02-18
### Added
- Publish a Linux ARM 32-bit binary with each release.
- Publish a `tmux-fastcopy-bin` package to AUR.

## 0.7.0 - 2022-02-10
### Changed
- Handle wrapping of long lines by Tmux.
  These lines will now be joined when copied.

## 0.6.2 - 2021-12-29
### Fixed
- Better handle single-quoted strings in Tmux configuration.

## 0.6.1 - 2021-10-28
### Fixed
- Homebrew formula: Conform to new Homebrew requirements.

## 0.6.0 - 2021-09-13
### Added
- Publish Homebrew formulae for the project.

## 0.5.1 - 2021-09-09
### Fixed
- Don't consume 100% CPU when idling.

## 0.5.0 - 2021-09-07
### Changed
- Change default action to `tmux load-buffer -`. This eliminates risk of
  hitting ARG_MAX with `set-buffer`--however unlikely that was.

### Fixed
- [(#38)]: Fix infinite loop when there's a single match.

  [(#38)]: https://github.com/abhinav/tmux-fastcopy/issues/38

## 0.4.0 - 2021-09-06
Highlight: The minimum required version of Tmux was lowered to 3.0.

### Added
- Add back `-log` flag. Use this flag to specify the destination for log
  messages.
- Add `-tmux` flag to specify the location of the tmux executable.

### Changed
- Support Tmux 3.0. Previously, tmux-fastcopy required at least Tmux 3.2.

## 0.3.1 - 2021-08-23
### Fixed
- Make path regex more accurate and avoid matching URLs.

## 0.3.0 - 2021-08-21
This release includes support for customizing regular expressions used by
tmux-fastcopy with the [`@fastcopy-regex-*` options][]. Check out the README
for more details.

### Added
- Support defining custom regular expressions, and overwriting or removing the
  default regular expressions.

### Removed
- Remove `-log` flag in favor of controlling the log file with an environment
  variable.

### Fixed
- Fix crashes of the wrapped binary not getting logged.

  [`@fastcopy-regex-*` options]: https://github.com/abhinav/tmux-fastcopy#fastcopy-regex-

## 0.2.0 - 2021-08-15
### Changed
- Download pre-built binary from GitHub if Go isn't available.

## 0.1.0 - 2021-08-14

- Initial release.
