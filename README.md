# rpaste

**rpaste** is a command-line client for [Rustypaste](https://github.com/orhun/rustypaste),
a self-hosted pastebin service.

## Usage

```none
usage: rpaste [-h] [-1] [-e TIME] [-f FILENAME] [-I] [-r URL] [-u URL] [-v]
              [-x SUFFIX]
              [file ...]

positional arguments:
  file         file to upload

options:
  -h, --help   show this help message and exit
  -1           one-shot upload
  -e TIME      expiration time
  -f FILENAME  custom filename
  -I           no Unix-time id suffix
  -r URL       remote source URL
  -u URL       URL to shorten
  -v           verbose mode
  -x SUFFIX    file suffix to add (including the ".")
```

## Requirements

- Optional: a command-line password manager like [pass](https://en.wikipedia.org/wiki/Pass_(software)) to store the authentication token
- Either of the following:
  - [uv](https://docs.astral.sh/uv/) (recommended)
  - Python 3.10 or later with two packages:
    - [HTTPX](https://python-httpx.org/))
    - [tomli](https://github.com/hukkin/tomli) for Python < 3.11

## Installation

1. Install the dependencies
2. Clone this repository.
3. Copy `rpaste.py` as `rpaste` to a directory in `PATH` (for example, `~/.local/bin/`).
   Change the `#!` line if you are not going to use uv.
4. Optional: store the API token in a command-line password manager
5. Create a configuration file in `~/.config/rpaste/config.toml` with the following contents:

```toml
url = "<your Rustypaste URL>"

# Either
token = "<your Rustypaste token>"
# or
token-command = "<command to get the token>"
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
