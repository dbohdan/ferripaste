# rpaste

**rpaste** is an alternative command-line client for [Rustypaste](https://github.com/orhun/rustypaste),
a self-hosted pastebin service.
rpaste offers different features from the official client [rustypaste-cli](https://github.com/orhun/rustypaste-cli):

- Basic password manager integration.
  rpaste can retrieve the authentication token from a command-line password manager like [pass](https://www.passwordstore.org/) by running a command.
- A custom file naming scheme.
  rpaste generates unique filenames by adding Unix timestamps before the file extension.
  This is performed independently of the Rustypaste server.
  For example, `foo.tar.gz` becomes `foo.1736864775.tar.gz`.

## Other features

- Supports expiring and one-shot uploads
- Verifies non-one-shot uploads without full download
- URL shortening
- Remote URL uploads

## Usage

```none
usage: rpaste [-h] [-1] [-c PATH] [-e TIME] [-f FILENAME] [-I] [-r URL]
              [-u URL] [-v] [-x SUFFIX]
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
  -u URL       URL to shorten
  -v           verbose mode
  -x SUFFIX    file suffix to add (including the ".")
```

## Requirements

- Either of the following:
  - [uv](https://docs.astral.sh/uv/) (recommended)
  - Python 3.11 or later with the package [HTTPX](https://python-httpx.org/)
- Optional: a command-line password manager like [pass](https://en.wikipedia.org/wiki/Pass_(software)) to store the authentication token

## Installation

1. Install the dependencies
2. Clone this repository
3. Copy `rpaste.py` as `rpaste` to a directory in `PATH` (for example, `~/.local/bin/`).
   Change the `#!` line if you are not going to use uv.
4. Optional: store the API token in a command-line password manager
5. Create a configuration file in `~/.config/rpaste/config.toml` with the following contents:

```toml
# Your Rustypaste URL:
url = "https://paste.example.com"

# One of the two token options:
# 1. Your literal token.
token = "foo123"
# 2. The command to get the token.
token-command = "pass show paste.example.com"
```

## Examples

```shell
# Upload a file.
rpaste file.txt

# Upload with custom name.
rpaste -f custom.txt file.txt

# Upload multiple files.
rpaste file1.txt file2.txt file3.txt

# Create a one-shot upload.
rpaste -1 file.txt

# Shorten a URL.
rpaste -u https://example.com

# Upload from a remote URL.
rpaste -r https://example.com/file.txt

# Set expiration time.
rpaste -e 1h file.txt
```

## License

MIT.
See the [LICENSE](LICENSE) file for details.
