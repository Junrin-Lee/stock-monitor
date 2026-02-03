# Homebrew Formula Directory

This directory contains the Homebrew formula for Stock Monitor, automatically managed by GoReleaser.

## Installation

Users can install Stock Monitor via Homebrew using:

```bash
# Method 1: Tap the repository
brew tap Junrin-Lee/stock-monitor https://github.com/Junrin-Lee/stock-monitor.git
brew install stock-monitor

# Method 2: Direct install from URL (no tap needed)
brew install --formula https://raw.githubusercontent.com/Junrin-Lee/stock-monitor/master/Formula/stock-monitor.rb
```

## Formula Management

The `stock-monitor.rb` formula file in this directory is automatically updated by GoReleaser during the release process. **Do not manually edit this file.**

When a new version is released:
1. GoReleaser builds the binaries
2. Uploads artifacts to GitHub Releases
3. Updates the formula with new version and SHA256 checksums
4. Commits the changes back to this repository

## Upgrading

Users can upgrade to the latest version:

```bash
brew upgrade stock-monitor
```

## Uninstalling

```bash
brew uninstall stock-monitor
brew untap Junrin-Lee/stock-monitor  # Optional: remove the tap
```
