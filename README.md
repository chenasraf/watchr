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
- **Config files**: YAML, TOML, or JSON config files for persistent settings
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
  -c, --config string              Load config from specified path
  -C, --show-config                Show loaded configuration and exit
  -r, --refresh int                Auto-refresh interval in seconds (0 = disabled)
  -p, --prompt string              Prompt string (default "watchr> ")
  -s, --shell string               Shell to use for executing commands (default "sh")
  -n, --no-line-numbers            Disable line numbers
  -w, --line-width int             Line number width (default 6)
  -P, --preview-size string        Preview size: number for lines/cols, or number% for percentage (default "40%")
  -o, --preview-position string    Preview position: bottom, top, left, right (default "bottom")
```

---

## 📁 Configuration File

`watchr` supports configuration files in YAML, TOML, or JSON format. Settings in config files serve as defaults that can be overridden by command-line flags.

### Config File Locations

Config files are searched in the following order (later files override earlier ones):

1. **XDG config directory** (Linux/macOS): `~/.config/watchr/watchr.{yaml,toml,json}`
2. **Windows**: `%APPDATA%\watchr\watchr.{yaml,toml,json}`
3. **Current directory** (project-local): `./watchr.{yaml,toml,json}`

### Example Configurations

**YAML** (`watchr.yaml`):
```yaml
shell: bash
preview-size: "50%"
preview-position: right
line-numbers: true
line-width: 4
prompt: "> "
refresh: 0
```

**TOML** (`watchr.toml`):
```toml
shell = "bash"
preview-size = "50%"
preview-position = "right"
line-numbers = true
line-width = 4
prompt = "> "
refresh = 0
```

**JSON** (`watchr.json`):
```json
{
  "shell": "bash",
  "preview-size": "50%",
  "preview-position": "right",
  "line-numbers": true,
  "line-width": 4,
  "prompt": "> ",
  "refresh": 0
}
```

### Priority Order

Configuration values are applied in this order (later sources override earlier ones):

1. Built-in defaults
2. XDG/system config file
3. Project-local config file (current directory)
4. Command-line flags

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
| `y`                | Yank (copy) selected line       |
| `?`                | Show help overlay               |

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
