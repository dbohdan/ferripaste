# Ferripaste

**Ferripaste** is an alternative command-line client for [Rustypaste](https://github.com/orhun/rustypaste),
a self-hosted pastebin service.
Ferripaste offers different features from the official client [rustypaste-cli](https://github.com/orhun/rustypaste-cli):

- Basic password manager integration.
  Ferripaste can retrieve the authentication token from a command-line password manager like [`pass`](https://www.passwordstore.org/) or the developer's [pago](https://github.com/dbohdan/pago) by running a command.
- A custom file naming scheme.
  Ferripaste generates unique filenames by adding a Unix timestamp before the file extension.
  For example, `foo.tar.gz` becomes `foo.1736864775.tar.gz`.
  This is performed independently of the Rustypaste server.
- [Exif](https://en.wikipedia.org/wiki/Exif) metadata removal from JPEG and PNG images

## Other features

- Expiring and one-shot uploads
- Verify non-one-shot uploads without full download
- URL shortening
- Remote URL uploads

## Usage

```none
Usage: ferri [options] files...

Options:
  -1, --one-shot        One-shot upload
  -I, --no-id           No Unix time ID suffix
  -c, --config string   Path to config file
  -e, --expire string   Expiration time
  -f, --filename string Custom filename
  -h, --help            Help
  -r, --remote string   Remote source URL
  -s, --strip-exif      Strip Exif metadata from JPEG and PNG
  -u, --url string      URL to shorten
  -v, --verbose int     Verbose mode
  -x, --ext string      File suffix to add (including the ".")

Arguments:
  files     Files to upload
```

## Requirements

- Go 1.22 or later
- Optional: a command-line password manager like `pass` to store the authentication token

## Installation

1. Install the dependencies: Go and, optionally, a CLI password manager
2. Install Ferripaste:

```shell
go install dbohdan.com/ferripaste/cmd/ferri@latest
```

Or clone the repoistory and build from source:

```shell
git clone https://github.com/dbohdan/ferripaste
cd ferripaste/
go build -C cmd/ferri/ -trimpath

# Install for the current user.
# You may need to add `~/.local/bin/` to `PATH`.
mkdir -p ~/.local/bin/
cp cmd/ferri/ferri ~/.local/bin/

# Install for all users on a system with sudo.
sudo install cmd/ferri/ferri /usr/local/bin/
```

3. Optional: store the Rustypaste authorization token in the command-line password manager
4. Create a configuration file (`~/.config/ferripaste/config.toml` on Linux and BSD) with the following contents:

```toml
# Your Rustypaste URL:
url = "https://paste.example.com"

# One of the two token options:
# 1. A literal token.
token = "foo123"
# 2. A command to get the token.
token-command = "pass show paste.example.com"
```

## Examples

```shell
# Upload a file.
ferri file.txt

# Upload with a custom name.
ferri -f custom.txt file.txt

# Upload multiple files.
ferri file1.txt file2.txt file3.txt

# Create a one-shot upload.
ferri -1 file.txt

# Shorten a URL.
ferri -u https://example.com

# Upload from a remote URL.
ferri -r https://example.com/file.txt

# Set expiration time.
ferri -e 1h file.txt
```

## License

MIT.
See the [`LICENSE`](LICENSE) file for details.
