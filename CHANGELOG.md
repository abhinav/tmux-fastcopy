# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Changed
- Change default action to `tmux load-buffer -`. This eliminates risk of
  hitting ARG_MAX with `set-buffer`--however unlikely that was.

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
