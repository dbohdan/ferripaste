#! /usr/bin/env python3

from __future__ import annotations

import argparse
import asyncio
import re
import shlex
import shutil
import subprocess as sp
import sys
import tempfile
import time
import tomllib
import traceback
from dataclasses import dataclass
from pathlib import Path
from typing import IO, TYPE_CHECKING, Any

import httpx
import loguru
from loguru import logger
from xdg_base_dirs import xdg_config_home

AUTHZ_HEADER = "authorization"
FORMAT_MAX_BODY_LEN = 512
CONFIG_FILE = xdg_config_home() / "ferripaste/config.toml"
TIMEOUT = 30

if TYPE_CHECKING:
    ReqFiles = list[str] | list[tuple[str, IO[bytes], str]]


@dataclass(frozen=True)
class Upload:
    name: str
    status: int
    url: str


def main() -> None:
    asyncio.run(run())


async def run() -> None:
    args = cli()
    conf = config(args.config)

    logger.configure(
        handlers=[
            {
                "sink": sys.stderr,
                "format": format_log,
            },
        ],
    )

    dests = []
    upload_id = str(int(time.time()))

    if args.files:
        if args.filename:
            dests = [args.filename]
        else:
            for file in args.files:
                stem, dot, ext = file.name.partition(".")

                dest = re.sub(r"[\s%]", "-", stem)
                if not args.no_id:
                    dest = dest + "." + upload_id

                dest += dot + ext + args.suffix

                dests.append(dest)

    headers = {AUTHZ_HEADER: conf["token"]}

    field = ""
    names = []
    req_files: ReqFiles = []
    temp_dir = None

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
            names = processed_files = args.files[:]

            if args.strip_exif:
                temp_dir = tempfile.TemporaryDirectory()

                processed_files = [
                    copy_without_exif(file, Path(temp_dir.name)) or file
                    for file in args.files
                ]

            req_files = [
                (dest, file.open("rb"), "application/octet-stream")
                for file, dest in zip(processed_files, dests, strict=True)
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

        sys.stdout.write(
            "\n".join(upload.url for upload in uploads if upload.url) + "\n"
        )
        exit_code = (
            0 if all(upload.status == httpx.codes.OK for upload in uploads) else 1
        )

        # Verify the uploads.
        if not args.one_shot and not verify_uploads(uploads):
            exit_code = 1

        sys.exit(exit_code)

    except (FileNotFoundError, httpx.RequestError, sp.CalledProcessError) as e:
        if args.verbose:
            logger.error(traceback.format_exc())
        else:
            logger.error(str(e))

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
        for req in reqs:
            logger.debug(format_request(req))

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
            logger.error(f"{upload.name} failed to upload with status {upload.status}")

            continue

        with httpx.stream(method="GET", timeout=TIMEOUT, url=upload.url) as resp:
            if resp.status_code != httpx.codes.OK:
                result = False
                logger.error(
                    f"{upload.url} failed URL verification "
                    f"with status {resp.status_code}",
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
        "--one-shot",
        action="store_true",
        dest="one_shot",
        help="one-shot upload",
    )

    parser.add_argument(
        "-c",
        "--config",
        dest="config",
        help="path to config file",
        metavar="PATH",
        type=Path,
    )

    parser.add_argument(
        "-e",
        "--expire",
        dest="expire_time",
        help="expiration time",
        metavar="TIME",
    )

    fn_group.add_argument(
        "-f",
        "--filename",
        dest="filename",
        help="custom filename",
        metavar="FILENAME",
    )

    parser.add_argument(
        "-I",
        "--no-id",
        action="store_true",
        dest="no_id",
        help="no Unix time ID suffix",
    )

    parser.add_argument(
        "-r",
        "--remote",
        dest="remote_url",
        help="remote source URL",
        metavar="URL",
    )

    parser.add_argument(
        "-s",
        "--strip-exif",
        action="store_true",
        dest="strip_exif",
        help="strip Exif metadata",
    )

    parser.add_argument(
        "-u",
        "--url",
        dest="url_to_shorten",
        help="URL to shorten",
        metavar="URL",
    )

    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        dest="verbose",
        help="verbose mode",
    )

    fn_group.add_argument(
        "-x",
        "--ext",
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


def config(config_path: Path | None = None) -> dict[str, Any]:
    path = config_path if config_path else CONFIG_FILE
    config = tomllib.loads(path.read_text())

    if "token" not in config:
        config["token"] = sp.run(
            shlex.split(config["token-command"]),
            capture_output=True,
            check=True,
            text=True,
        ).stdout.rstrip()

    return config


def format_log(record: loguru.Record) -> str:
    return f"[{record['level'].name.lower()}] {record['message']}\n"


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


def copy_without_exif(src: Path, dest_dir: Path) -> Path:
    dest_subdir = dest_dir

    i = 1
    while dest_subdir.is_dir():
        dest_subdir = dest_dir / str(i)
        i += 1

    dest = dest_subdir / src.name
    dest_subdir.mkdir()

    shutil.copy(src, dest)
    sp.run(["exiftool", "-all=", "-quiet", dest], check=True)

    return dest


if __name__ == "__main__":
    main()
