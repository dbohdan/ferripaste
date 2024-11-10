#! /usr/bin/env -S pipx run
# /// script
# dependencies = [
#   "httpx<2",
#   "tomli==2.*",
# ]
# requires-python = ">=3.10"
# ///

from __future__ import annotations

import argparse
import asyncio
import re
import subprocess as sp
import sys
import time
import traceback
from dataclasses import dataclass
from pathlib import Path
from typing import Any, IO, TYPE_CHECKING

import httpx

try:
    import tomllib
except ModuleNotFoundError:
    import tomli as tomllib

AUTHZ_HEADER = "authorization"
FORMAT_MAX_BODY_LEN = 512
CONFIG_FILE = Path.home() / ".config/rpaste/config.toml"
TIMEOUT = 30

if TYPE_CHECKING:
    ReqFiles = list[str] | list[tuple[str, IO[bytes], str]]


@dataclass(frozen=True)
class Upload:
    name: str
    status: int
    url: str


async def main() -> None:
    args = cli()
    conf = config()

    dests = []
    upload_id = str(int(time.time()))

    if args.files:
        if args.filename:
            dests = [args.filename]
        else:
            for file in args.files:
                ext = file.suffix
                dest = re.sub(r"[\s%]", "-", file.stem)
                if not args.no_id:
                    dest = f"{dest}.{upload_id}"
                dest += ext + args.suffix

                dests.append(dest)

    headers = {AUTHZ_HEADER: conf["token"]}

    field = ""
    names = []
    req_files: ReqFiles = []

    if args.expire_time:
        headers["expire"] = args.expire_time

    try:
        if args.remote_url:
            field = "remote"
            names = req_files = [args.remote_url]
        elif args.url_to_shorten:
            field = "oneshot_url" if args.one_shot else "url"
            names = req_files = [args.url_to_shorten]
        else:
            field = "oneshot" if args.one_shot else "file"
            names = args.files[:]
            req_files = [
                (dest, file.open("rb"), "application/octet-stream")
                for file, dest in zip(args.files, dests, strict=True)
            ]

        # Upload.
        client = httpx.AsyncClient(
            timeout=TIMEOUT,
        )

        uploads = await upload_files(
            client,
            conf["url"],
            headers,
            field,
            req_files,
            names=names,
            verbose=args.verbose,
        )

        print("\n".join(upload.url for upload in uploads if upload.url))
        exit_code = (
            0 if all(upload.status == httpx.codes.OK for upload in uploads) else 1
        )

        # Verify the uploads.
        if not args.one_shot and not verify_uploads(uploads):
            exit_code = 1

        sys.exit(exit_code)

    except (FileNotFoundError, httpx.RequestError) as e:
        if args.verbose:
            print(traceback.format_exc(), end="", file=sys.stderr)
        else:
            print(f"error: {e}", file=sys.stderr)

        sys.exit(1)


async def upload_files(
    client: httpx.AsyncClient,
    url: str,
    headers: dict[str, str],
    field: str,
    files: ReqFiles,
    *,
    names: list[str],
    verbose: bool,
) -> list[Upload]:
    reqs = [
        client.build_request(
            "POST",
            url,
            headers=headers,
            files={field: file},
        )
        for file in files
    ]

    if verbose:
        print(
            "\n\n".join(format_request(req) for req in reqs),
            file=sys.stderr,
        )

    resps = await asyncio.gather(
        *[client.send(req) for req in reqs],
    )

    return [
        Upload(name, resp.status_code, resp.text.rstrip())
        for name, resp in zip(names, resps, strict=True)
    ]


def verify_uploads(uploads: list[Upload]) -> bool:
    result = True

    for upload in uploads:
        if not upload.url:
            result = False
            print(
                f"error: {upload.name} failed to upload "
                f"with status {upload.status}",
                file=sys.stderr,
            )

            continue

        with httpx.stream(method="GET", timeout=TIMEOUT, url=upload.url) as resp:
            if resp.status_code != httpx.codes.OK:
                result = False
                print(
                    f"error: {upload.url} failed URL verification with "
                    f"status {resp.status_code}",
                    file=sys.stderr,
                )

    return result


def cli() -> argparse.Namespace:
    parser = argparse.ArgumentParser()

    fn_group = parser.add_mutually_exclusive_group()

    parser.add_argument(
        "files",
        help="file to upload",
        metavar="file",
        nargs="*",
        type=Path,
    )

    parser.add_argument(
        "-1",
        action="store_true",
        dest="one_shot",
        help="one-shot upload",
    )

    parser.add_argument(
        "-e",
        dest="expire_time",
        help="expiration time",
        metavar="TIME",
    )

    fn_group.add_argument(
        "-f",
        dest="filename",
        help="custom filename",
        metavar="FILENAME",
    )

    parser.add_argument(
        "-I",
        action="store_true",
        dest="no_id",
        help="no Unix-time id suffix",
    )

    parser.add_argument(
        "-r",
        dest="remote_url",
        help="remote source URL",
        metavar="URL",
    )

    parser.add_argument(
        "-u",
        dest="url_to_shorten",
        help="URL to shorten",
        metavar="URL",
    )

    parser.add_argument("-v", action="store_true", dest="verbose", help="verbose mode")

    fn_group.add_argument(
        "-x",
        default="",
        dest="suffix",
        help='file suffix to add (including the ".")',
        metavar="SUFFIX",
    )

    args = parser.parse_args()

    mutually_exclusive = (args.files, args.remote_url, args.url_to_shorten)
    if sum(int(bool(x)) for x in mutually_exclusive) != 1:
        parser.error("one or more file arguments, -r, or -u required")

    if args.filename:
        if args.no_id:
            parser.error("argument -I: not allowed with argument -f")

        if len(args.files) > 1:
            parser.error("argument -f: not allowed with more than one file")

    if args.one_shot and args.remote_url:
        parser.error(
            "argument -1: not allowed with argument -r",
        )

    return args


def config() -> dict[str, Any]:
    config = tomllib.loads(CONFIG_FILE.read_text())

    config["token"] = sp.run(
        ["pass", "show", config["pass-name"]],
        capture_output=True,
        check=True,
        text=True,
    ).stdout.rstrip()

    return config


def format_request(req: httpx.Request) -> str:
    def format_header(header: str, value: str) -> str:
        if header == AUTHZ_HEADER:
            value = "***"
        elif header == "content-length":
            value = f"{value}"

        return f"{header}: {value}"

    req.read()

    headers = (format_header(header, value) for header, value in req.headers.items())

    body = "(empty body)"
    if req.content:
        body = (
            str(req.content)
            if len(req.content) <= FORMAT_MAX_BODY_LEN
            else "(body omitted for length)"
        )

    req_info = [
        "Request:",
        str(req.url),
        "",
        *headers,
        "",
        body,
    ]

    return "\n    ".join(req_info)


if __name__ == "__main__":
    asyncio.run(main())
