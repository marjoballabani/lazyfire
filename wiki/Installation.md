# Installation

## Prerequisites

### Firebase CLI

LazyFire requires the Firebase CLI to be installed and authenticated:

```bash
# Install Firebase CLI
npm install -g firebase-tools

# Login to Firebase
firebase login
```

### Nerd Font (Optional)

For icons to display correctly, install a [Nerd Font](https://www.nerdfonts.com/). Popular options:
- JetBrains Mono Nerd Font
- Fira Code Nerd Font
- Hack Nerd Font

## Install LazyFire

### Using Go

```bash
go install github.com/marjoballabani/lazyfire@latest
```

### From Source

```bash
git clone https://github.com/marjoballabani/lazyfire.git
cd lazyfire
go build -o lazyfire .
./lazyfire
```

### Using Homebrew (macOS)

```bash
brew install marjoballabani/tap/lazyfire
```

## Verify Installation

```bash
lazyfire --version
```

## First Run

Simply run `lazyfire` in your terminal. It will:
1. Detect your Firebase authentication
2. Load available projects
3. Display the TUI

```bash
lazyfire
```
