# Ferripaste

**Ferripaste** is an alternative command-line client for [Rustypaste](https://github.com/orhun/rustypaste),
a self-hosted pastebin service.
Ferripaste offers different features from the official client [rustypaste-cli](https://github.com/orhun/rustypaste-cli):

- Basic password manager integration.
  Ferripaste can retrieve the authentication token from a command-line password manager like [pass](https://www.passwordstore.org/) by running a command.
- A custom file naming scheme.
  Ferripaste generates unique filenames by adding Unix timestamps before the file extension.
  This is performed independently of the Rustypaste server.
  For example, `foo.tar.gz` becomes `foo.1736864775.tar.gz`.
- [Exif](https://en.wikipedia.org/wiki/Exif) metadata removal from images using [ExifTool](https://en.wikipedia.org/wiki/ExifTool)

## Other features

- Supports expiring and one-shot uploads
- Verifies non-one-shot uploads without full download
- URL shortening
- Remote URL uploads

## Usage

```none
usage: ferripaste [-h] [-1] [-c PATH] [-e TIME] [-f FILENAME] [-I] [-r URL]
                  [-s] [-u URL] [-v] [-x SUFFIX]
                  [file ...]

positional arguments:
  file         file to upload

options:
  -h, --help   show this help message and exit
  -1           one-shot upload
  -c PATH      path to config file
  -e TIME      expiration time
  -f FILENAME  custom filename
  -I           no Unix-time id suffix
  -r URL       remote source URL
  -s           strip Exif metadata from images
  -u URL       URL to shorten
  -v           verbose mode
  -x SUFFIX    file suffix to add (including the ".")
```

## Requirements

- One of the following:
  - [uv](https://docs.astral.sh/uv/) (recommended)
  - [pipx](https://pipx.pypa.io/)
  - Python 3.11 or later with the package [HTTPX](https://python-httpx.org/)
- Optional: a command-line password manager like [pass](https://en.wikipedia.org/wiki/Pass_(software)) to store the authentication token
- Optional: ExifTool for Exif metadata removal

## Installation

The recommended way to install Ferripaste is with uv.

- Install the dependencies: uv, a CLI password manager (optional), ExifTool (optional)
- Clone this repository
- Install Ferripaste:

```shell
uv tool install --python 3.11 git+https://github.com/dbohdan/ferripaste@master
```

- Optional: store the Rustypaste authorization token in the command-line password manager
- Create a configuration file (`~/.config/ferripaste/config.toml` on Linux and BSD) with the following contents:

```toml
# Your Rustypaste URL:
url = "https://paste.example.com"

# One of the two token options:
# 1. The literal token.
token = "foo123"
# 2. The command to get the token.
token-command = "pass show paste.example.com"
```

## Examples

```shell
# Upload a file.
ferripaste file.txt

# Upload with custom name.
ferripaste -f custom.txt file.txt

# Upload multiple files.
ferripaste file1.txt file2.txt file3.txt

# Create a one-shot upload.
ferripaste -1 file.txt

# Shorten a URL.
ferripaste -u https://example.com

# Upload from a remote URL.
ferripaste -r https://example.com/file.txt

# Set expiration time.
ferripaste -e 1h file.txt
```

## License

MIT.
See the [LICENSE](LICENSE) file for details.
