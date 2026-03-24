# watchr

**`watchr`** is a terminal UI for running commands and interactively browsing their output. It
provides vim-style navigation, filtering, and a preview pane—all without leaving your terminal.

![Release](https://img.shields.io/github/v/release/chenasraf/watchr)
![Downloads](https://img.shields.io/github/downloads/chenasraf/watchr/total)
![License](https://img.shields.io/github/license/chenasraf/watchr)

![Promo](https://github.com/user-attachments/assets/ec5ab94b-ef91-40d8-a604-9047212a8faf)

---

## 🚀 Features

- **Interactive output viewer**: Browse command output with vim-style keybindings
- **Live filtering**: Press `/` to filter output lines in real-time, with regex support (`//`)
- **Preview pane**: Toggle a resizable preview panel (bottom, top, left, or right) with JSON syntax
  highlighting
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

# Refresh every 500 milliseconds
watchr -r 500ms "date"

# Refresh every 1.5 seconds
watchr -r 1.5s "kubectl get pods"

# Refresh every 5 minutes
watchr -r 5m "df -h"

# Refresh every hour
watchr -r 1h "curl -s https://api.example.com/status"

# Watch file changes
watchr -r 5 "find . -name '*.go' -mmin -1"
```

### Options

```
Usage: watchr [options] <command to run>

Options:
  -c, --config string             Load config from specified path
  -h, --help                      Show help
  -i, --interactive               Run shell in interactive mode (sources ~/.bashrc, ~/.zshrc, etc.)
  -w, --line-width int            Line number width (default 6)
  -n, --no-line-numbers           Disable line numbers
  -o, --preview-position string   Preview position: bottom, top, left, right (default "bottom")
  -P, --preview-size string       Preview size: number for lines/cols, or number% for percentage (e.g., 10 or 40%) (default "40%")
  -p, --prompt string             Prompt string (default "watchr> ")
  -r, --refresh string            Auto-refresh interval (e.g., 1, 1.5, 500ms, 2s, 5m, 1h; default unit: seconds, 0 = disabled) (default "0")
      --refresh-from-start        Start refresh timer when command starts (default: when command ends)
  -s, --shell string              Shell to use for executing commands (default "sh")
  -C, --show-config               Show loaded configuration and exit
  -v, --version                   Show version
```

---

## 📁 Configuration File

`watchr` supports configuration files in YAML, TOML, or JSON format. Settings in config files serve
as defaults that can be overridden by command-line flags.

### Config File Locations

Config files are searched in the following order (later files override earlier ones):

1. **XDG config directory** (Linux/macOS): `~/.config/watchr/watchr.{yaml,toml,json}`
2. **Windows**: `%APPDATA%\watchr\watchr.{yaml,toml,json}`
3. **Current directory** (project-local): `./watchr.{yaml,toml,json}`

### Example Configurations

**YAML** (`watchr.yaml`):

```yaml
shell: bash
preview-size: '50%'
preview-position: right
line-numbers: true
line-width: 4
prompt: '> '
refresh: 0 # disabled; or use: 2, 1.5, "500ms", "2s", "5m", "1h"
interactive: false
```

**TOML** (`watchr.toml`):

```toml
shell = "bash"
preview-size = "50%"
preview-position = "right"
line-numbers = true
line-width = 4
prompt = "> "
refresh = 0 # disabled; or use: 2, 1.5, "500ms", "2s", "5m", "1h"
interactive = false
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
  "refresh": 0,
  "interactive": false
}
```

The `refresh` option accepts:

- Numbers: `2` or `1.5` (interpreted as seconds)
- Explicit units: `"500ms"`, `"2s"`, `"5m"`, `"1h"`

### Priority Order

Configuration values are applied in this order (later sources override earlier ones):

1. Built-in defaults
2. XDG/system config file
3. Project-local config file (current directory)
4. Command-line flags

---

## ⌨️ Keybindings

| Key                | Action                           |
| ------------------ | -------------------------------- |
| `r`, `Ctrl-r`      | Reload (re-run command)          |
| `R`                | Reload & clear all lines         |
| `Del`              | Delete selected line             |
| `Ctrl-Del`         | Clear all lines (with confirm)   |
| `c`                | Stop running command             |
| `q`, `Esc`         | Quit                             |
| `j`, `k`           | Move down/up                     |
| `g`                | Go to first line                 |
| `G`                | Go to last line                  |
| `Ctrl-d`, `Ctrl-u` | Half page down/up                |
| `PgDn`, `Ctrl-f`   | Full page down                   |
| `PgUp`, `Ctrl-b`   | Full page up                     |
| `p`                | Toggle preview pane              |
| `+` / `-`          | Increase / decrease preview size |
| `J` / `K`          | Scroll preview down / up         |
| `/`                | Enter filter mode                |
| `//`               | Toggle regex filter mode         |
| `Esc`              | Exit filter mode / clear filter  |
| `y`                | Yank (copy) selected line        |
| `Y`                | Yank selected line (plain text)  |
| `:`                | Open command palette             |
| `?`                | Show help overlay                |

### Filter mode

When in filter mode (`/`), the following keys are available:

| Key                      | Action                                   |
| ------------------------ | ---------------------------------------- |
| `Enter`                  | Confirm filter                           |
| `Esc`                    | Cancel and clear filter                  |
| `Left` / `Right`         | Move cursor within filter                |
| `Alt-Left` / `Alt-Right` | Move cursor by word                      |
| `Backspace`              | Delete character before cursor           |
| `Alt-Backspace`          | Delete word before cursor                |
| `/`                      | Toggle regex mode (when filter is empty) |

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
