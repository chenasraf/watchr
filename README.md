# watchr

**`watchr`** is a terminal UI for running commands and interactively browsing their output. It
provides vim-style navigation, filtering, and a preview pane—all without leaving your terminal.

![Release](https://img.shields.io/github/v/release/chenasraf/watchr)
![Downloads](https://img.shields.io/github/downloads/chenasraf/watchr/total)
![License](https://img.shields.io/github/license/chenasraf/watchr)

---

## 🚀 Features

- **Interactive output viewer**: Browse command output with vim-style keybindings
- **Live filtering**: Press `/` to filter output lines in real-time
- **Preview pane**: Toggle a preview panel (bottom, top, left, or right)
- **Auto-refresh**: Optionally re-run commands at specified intervals
- **Line numbers**: Optional line numbering with configurable width
- **Full-screen TUI**: Clean, distraction-free interface using your entire terminal

---

## 🎯 Installation

### Download Precompiled Binaries

Grab the latest release for **Linux**, **macOS**, or **Windows**:

- [Releases →](https://github.com/chenasraf/watchr/releases/latest)

### Homebrew (macOS/Linux)

Install directly from the tap:

```bash
brew install chenasraf/tap/watchr
```

Or tap and then install the package:

```bash
brew tap chenasraf/tap
brew install watchr
```

### From Source

```bash
git clone https://github.com/chenasraf/watchr
cd watchr
make build
make install  # installs to ~/.local/bin
```

---

## ✨ Getting Started

### Basic Usage

```bash
# View output of any command
watchr ls -la

# View logs
watchr "tail -100 /var/log/system.log"

# Monitor processes
watchr "ps aux"
```

### Auto-Refresh

```bash
# Refresh every 2 seconds
watchr -r 2 "docker ps"

# Watch file changes
watchr -r 5 "find . -name '*.go' -mmin -1"
```

### Options

```
Usage: watchr [options] <command to run>

Options:
  -h, --help                       Show help
  -v, --version                    Show version
  -r, --refresh int                Auto-refresh interval in seconds (0 = disabled)
  -p, --prompt string              Prompt string (default "watchr> ")
  -s, --shell string               Shell to use for executing commands (default "sh")
  -n, --no-line-numbers            Disable line numbers
  -w, --line-width int             Line number width (default 6)
  -P, --preview-size string        Preview size: number for lines/cols, or number% for percentage (default "40%")
  -o, --preview-position string    Preview position: bottom, top, left, right (default "bottom")
```

---

## ⌨️ Keybindings

| Key                | Action                          |
| ------------------ | ------------------------------- |
| `r`, `Ctrl-r`      | Reload (re-run command)         |
| `q`, `Esc`         | Quit                            |
| `j`, `k`           | Move down/up                    |
| `g`                | Go to first line                |
| `G`                | Go to last line                 |
| `Ctrl-d`, `Ctrl-u` | Half page down/up               |
| `PgDn`, `Ctrl-f`   | Full page down                  |
| `PgUp`, `Ctrl-b`   | Full page up                    |
| `p`                | Toggle preview pane             |
| `/`                | Enter filter mode               |
| `Esc`              | Exit filter mode / clear filter |

---

## 🛠️ Contributing

I am developing this package on my free time, so any support, whether code, issues, or just stars is
very helpful to sustaining its life. If you are feeling incredibly generous and would like to donate
just a small amount to help sustain this project, I would be very very thankful!

<a href='https://ko-fi.com/casraf' target='_blank'>
<img height='36' style='border:0px;height:36px;' src='https://cdn.ko-fi.com/cdn/kofi1.png?v=3' alt='Buy Me a Coffee at ko-fi.com' />
</a>

I welcome any issues or pull requests on GitHub. If you find a bug, or would like a new feature,
don't hesitate to open an appropriate issue and I will do my best to reply promptly.

---

## 📜 License

`watchr` is licensed under the [MIT License](/LICENSE).
