# Installation

## Homebrew (Recommended)

The easiest way to install LazyFire:

```bash
brew install marjoballabani/tap/lazyfire
```

## Using Go

If you have Go installed:

```bash
go install github.com/marjoballabani/lazyfire@latest
```

Make sure `$GOPATH/bin` is in your `PATH`.

## Download Binary

Download pre-built binaries from the [releases page](https://github.com/marjoballabani/lazyfire/releases).

### macOS

```bash
# Intel
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_darwin_amd64.tar.gz | tar xz
sudo mv lazyfire /usr/local/bin/

# Apple Silicon
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_darwin_arm64.tar.gz | tar xz
sudo mv lazyfire /usr/local/bin/
```

### Linux

```bash
# x86_64
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_linux_amd64.tar.gz | tar xz
sudo mv lazyfire /usr/local/bin/

# ARM64
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_linux_arm64.tar.gz | tar xz
sudo mv lazyfire /usr/local/bin/
```

### Windows

Download the `.zip` file from the releases page and extract `lazyfire.exe` to a directory in your `PATH`.

## Build from Source

```bash
git clone https://github.com/marjoballabani/lazyfire.git
cd lazyfire
go build -o lazyfire .
```

## Firebase Authentication

LazyFire uses your existing Firebase CLI credentials. Make sure you're logged in:

```bash
firebase login
```

To use a specific service account:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
lazyfire
```
