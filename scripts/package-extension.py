#!/usr/bin/env python3
import json
import re
import sys
import zipfile
from pathlib import Path


INCLUDE_FILES = [
    "manifest.json",
    "popup.html",
    "popup.js",
    "service-worker.js",
    "shared-config.js",
    "icons/icon-16.png",
    "icons/icon-32.png",
    "icons/icon-48.png",
    "icons/icon-128.png",
]


def package_slug(name):
    slug = re.sub(r"[^a-z0-9]+", "-", name.lower()).strip("-")
    return slug or "extension"


def main():
    repo_root = Path(__file__).resolve().parent.parent
    extension_dir = repo_root / "extension"
    build_dir = repo_root / "build"
    manifest_path = extension_dir / "manifest.json"

    if not manifest_path.is_file():
        print(f"error: manifest not found at {manifest_path}", file=sys.stderr)
        return 1

    with manifest_path.open(encoding="utf-8") as manifest_file:
        manifest = json.load(manifest_file)

    name = manifest.get("name", "extension")
    version = manifest.get("version", "0")
    zip_path = build_dir / f"{package_slug(name)}-{version}.zip"

    build_dir.mkdir(parents=True, exist_ok=True)

    try:
        with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for relative_path in INCLUDE_FILES:
                source = extension_dir / relative_path
                if not source.is_file():
                    raise FileNotFoundError(
                        f"required extension file is missing: {source}"
                    )
                archive.write(source, relative_path)
    except OSError as error:
        print(f"error: {error}", file=sys.stderr)
        return 1

    print(zip_path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
