# bhavyadang-tui

A terminal portfolio for [bhavyadang.in](https://bhavyadang.in), accessible via SSH.

```bash
ssh ssh.bhavyadang.in
```

Built with [Go](https://go.dev), [Wish](https://github.com/charmbracelet/wish), and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

---

# TODO:

- [ ] host it on rpi4
- [ ] update data

---

## Navigation

| Key                    | Action               |
| ---------------------- | -------------------- |
| `←` / `h` or `→` / `l` | Navigate tabs        |
| `1` – `5`              | Jump to tab directly |
| `j` / `↓`              | Scroll down          |
| `k` / `↑`              | Scroll up            |
| `q` / `Ctrl+C`         | Quit                 |

---

## Local Development

### Prerequisites

- Go 1.24+

### Run locally

```bash
git clone https://github.com/bhavya-dang/bhavyadang-tui
cd bhavyadang-tui

go mod tidy
go run .
```

Then in another terminal:

```bash
ssh localhost -p 23234
```

---

## Tech Stack

- **[Wish](https://github.com/charmbracelet/wish)** – SSH server framework for Go
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** – TUI framework (Elm architecture)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** – Terminal styling
- **[Go](https://go.dev)** – Because of course

---

MIT © Bhavya Dang 2026
