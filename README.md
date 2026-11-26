![Dinky Logo](logo_small.png)

# Dinky

> A little text editor that just works

> **This is a fork of [Dinky](https://github.com/sedwards2009/dinky) by [Simon Edwards (sedwards2009)](https://github.com/sedwards2009) — the original creator and maintainer of both Dinky and smidgen (the editing engine at its heart). A massive, heartfelt thank you; this project would not exist without his work. Extreme gratitude.** 💙
>
> Looking for the original/stable Dinky? See [sedwards2009/dinky](https://github.com/sedwards2009/dinky). This fork has diverged with goals I personally believed may not align with original creator.

Dinky is a terminal-based text editor designed for people who want to quickly edit text without having to learn the arcane ways of vi, emacs, or other traditional terminal editors. It wraps the excellent editor component from [micro](https://github.com/zyedidia/micro) with a familiar GUI-style interface complete with menus, dialogs, and intuitive navigation.

**GitHub Repository:** https://github.com/TechnoSparks/dinky

## Why Dinky?

Have you ever found yourself needing to quickly edit a file in the terminal but no wanting to waste brain cells learning another text editor "religion"? Then Dinky is for you! It provides a familiar interface with mouse support, menus, dialogs, scroll bars, and keyboard shortcuts. It all feels natural because you have been working like this on the desktop for years. It is like a desktop editor was squeezed down to fit inside your terminal.

Now you can edit that git commit or random config file in the terminal and then quickly get back to your desktop editor to do "real work". Dinky will even do a cheap cosplay impression of VSCode just to hammer on that sense of familiarity!

## Screenshots

![File menu screenshot](screenshot.png)

Cutting edge graphical technology from 1985 is now available to you in the year 2026! Wow! 🤩

![File dialog screenshot](screenshot2.png)

Yes! We can have nice things in a terminal application. 💎

![Settings dialog screenshot](screenshot3.png)

Settings but without having to hack some obscure config file. Folks, we're living in the future!

## Features

- **Single executable**: Just one executable to download, runs from anywhere
- **Mouse support**: Including scroll wheel
- **Multiple tabs**: Work with multiple files simultaneously
- **File dialog**: You know, a normal file dialog
- **Multi-cursor editing**: Add cursors for simultaneous editing
- **Undo/Redo**: Full undo history
- **Cut/Copy/Paste**: Standard clipboard operations. Choose in Settings → Clipboard whether to copy through the terminal (OSC 52, works over SSH, the default), the system clipboard, or keep text inside Dinky. Turn on Copy on Select there to also copy text selected with the mouse.
- **Desktop style keyboard shortcuts**: and menus which show the shortcuts so you can subconsciously learn them while you ~~sleep~~ work.
- **Go to Line**: Jump to specific lines & columns
- **Find**: Find that needle in the haystack
- **Find & Replace**: Replace said needle in haystack
- **Line numbers**: Toggle line number display
- **Bookmarks**: Keep track of positions in large files
- **Bracket matching**: Navigate between matching brackets
- **Soft wrap**: Wrap long lines to window width
- **Show whitespace**: You need this for Makefiles, sorry
- **Syntax highlighting**: Support for multiple programming languages, looks pretty ✨
- **Colour schemes**: Pick one in Settings → Theme, or add your own (see [Custom colour schemes](#custom-colour-schemes))
- **Vertical Ruler**: Keep those long lines under control
- **Sort & reverse lines**: Order those lines too
- **URL encode/decode, upper/lower case, format JSON**: Sometimes you just have to

## Custom colour schemes

Put colour scheme files ending in `.micro` in the `colorschemes` folder next to Dinky's `settings.json`:

- Linux: `~/.config/dinky/colorschemes/`
- macOS: `~/Library/Application Support/dinky/colorschemes/`

They use the same format as [micro's colour schemes](https://github.com/micro-editor/micro/blob/master/runtime/help/colors.md), so you can copy a micro colour scheme straight in. `include` lines are not supported. A file with the same name as a built-in scheme replaces it. New files show up the next time you open Settings. Lines that aren't valid are skipped, and the first problem is shown in the status bar when the scheme is applied. A file that exists but can't be read is reported the same way, and Dinky uses the built-in scheme of that name, or monokai if there is none.

## Installation

Download a release binary from the [releases page](https://github.com/TechnoSparks/dinky/releases).

Or install with a one-liner (latest release, or a specific PR's build):

```bash
# Latest release (or newest CI build until a release is published)
curl -fsSL https://raw.githubusercontent.com/TechnoSparks/dinky/main/install.sh | sh

# Build from a specific pull request
curl -fsSL https://raw.githubusercontent.com/TechnoSparks/dinky/main/install.sh | sh -s -- 10
```

The script installs into the current directory by default. Use `--install-dir` to
choose another location; it will use `sudo` when the directory is not writable.

Run `sh install.sh --help` for all options, including `--run-id`, `--repo`, `--dry-run`,
`-y/--yes`, and the `DINKY_PLATFORM` override (e.g. `DINKY_PLATFORM=linux-amd64`) for
forcing a download on an unsupported/undetected platform.

### or build from source

```bash
git clone https://github.com/TechnoSparks/dinky
cd dinky
go build dinky.go
```

Cross-compile for other platforms (static binaries):

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dinky-linux-amd64 .

# Linux arm32 (ARMv7)
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o dinky-linux-arm .

# Linux arm64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dinky-linux-arm64 .

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o dinky-darwin-arm64 .
```

## Usage

```bash
# Start with a new file
./dinky

# Open an existing file
./dinky filename.txt

# Open multiple files
./dinky file1.txt file2.txt
```

## Development Status

Very much in development, So much development. I'll try to do regular releases as things move along.

## Technical Details

Dinky is built in Go and uses:

- **[smidgen](https://github.com/sedwards2009/smidgen)**: The editor component (based on micro's editing engine)
- **[tview](https://github.com/rivo/tview/)**: Terminal UI framework
- **[tcell](https://github.com/gdamore/tcell)**: Cross-platform terminal handling

## License

Dinky is released under the [MIT License](LICENSE).

## Author

Created by **Simon Edwards** - [@sedwards2009](https://github.com/sedwards2009)

## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests on the [GitHub repository](https://github.com/TechnoSparks/dinky).

---

_Dinky: Because sometimes you just want to edit a file without having to join a cult._
