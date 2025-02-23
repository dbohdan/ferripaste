# Ferripaste

**Ferripaste** is an alternative command-line client for [Rustypaste](https://github.com/orhun/rustypaste),
a self-hosted pastebin service.
Ferripaste offers different features from the official client [rustypaste-cli](https://github.com/orhun/rustypaste-cli):

- Basic password manager integration.
  Ferripaste can retrieve the authentication token from a command-line password manager like [`pass`](https://www.passwordstore.org/) by running a command.
- A custom file naming scheme.
  Ferripaste generates unique filenames by adding a Unix timestamp before the file extension.
  For example, `foo.tar.gz` becomes `foo.1736864775.tar.gz`.
  This is performed independently of the Rustypaste server.
- [Exif](https://en.wikipedia.org/wiki/Exif) metadata removal from images using [ExifTool](https://en.wikipedia.org/wiki/ExifTool)

## Other features

- Supports expiring and one-shot uploads
- Verifies non-one-shot uploads without full download
- URL shortening
- Remote URL uploads

## Usage

```none
usage: ferri [-h] [-1] [-c PATH] [-e TIME] [-f FILENAME] [-I] [-r URL] [-s]
             [-u URL] [-v] [-x SUFFIX]
             [file ...]

positional arguments:
  file                  file to upload

options:
  -h, --help            show this help message and exit
  -1, --one-shot        one-shot upload
  -c, --config PATH     path to config file
  -e, --expire TIME     expiration time
  -f, --filename FILENAME
                        custom filename
  -I, --no-id           no Unix time ID suffix
  -r, --remote URL      remote source URL
  -s, --strip-exif      strip Exif metadata
  -u, --url URL         URL to shorten
  -v, --verbose         verbose mode
  -x, --ext SUFFIX      file suffix to add (including the ".")
```

## Requirements

- One of the following:
  - [uv](https://docs.astral.sh/uv/) (recommended)
  - [pipx](https://pipx.pypa.io/)
  - Python 3.11 or later with the package [HTTPX](https://www.python-httpx.org/)
- Optional: a command-line password manager like [`pass`](https://en.wikipedia.org/wiki/Pass_(software)) to store the authentication token
- Optional: ExifTool for Exif metadata removal

## Installation

The recommended way to install Ferripaste is with uv:

1. Install the dependencies: uv, a CLI password manager (optional), and ExifTool (optional)
2. Clone this repository
3. Install Ferripaste:

```shell
uv tool install --python 3.13 git+https://github.com/dbohdan/ferripaste@master
```

4. Optional: store the Rustypaste authorization token in the command-line password manager
5. Create a configuration file (`~/.config/ferripaste/config.toml` on Linux and BSD) with the following contents:

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
