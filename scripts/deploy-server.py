#!/usr/bin/env python3
import shlex
import subprocess
import sys
from pathlib import Path


def usage(script_path):
    print(f"usage: {script_path} <remote-host>", file=sys.stderr)


def latest_server_package(build_dir):
    packages = [
        package
        for package in build_dir.glob("history-server-*.pkg.tar.zst")
        if not package.name.startswith("history-server-debug-")
    ]
    if not packages:
        return None

    return max(packages, key=lambda package: (package.stat().st_mtime, package.name))


def build_server_package(repo_root):
    subprocess.run(
        ["bash", str(repo_root / "scripts" / "build-server.sh")],
        cwd=repo_root,
        check=True,
    )


def remote_install_command(remote_path):
    quoted_path = shlex.quote(remote_path)
    cleanup_command = f"rm -f -- {quoted_path}"
    install_command = f"sudo pacman -U --noconfirm {quoted_path}"
    restart_command = "sudo systemctl restart history-server"
    return (
        f"trap {shlex.quote(cleanup_command)} EXIT; "
        f"{install_command} && {restart_command}"
    )


def deploy_package(remote_host, package_path):
    remote_path = f"/tmp/{package_path.name}"

    subprocess.run(
        ["scp", str(package_path), f"{remote_host}:{remote_path}"],
        check=True,
    )
    subprocess.run(
        ["ssh", remote_host, remote_install_command(remote_path)],
        check=True,
    )


def main():
    if len(sys.argv) != 2:
        usage(sys.argv[0])
        return 1

    remote_host = sys.argv[1]
    repo_root = Path(__file__).resolve().parent.parent
    build_dir = repo_root / "build"

    try:
        build_server_package(repo_root)
    except subprocess.CalledProcessError as error:
        return error.returncode

    package_path = latest_server_package(build_dir)

    if package_path is None:
        print(
            f"error: no history-server package found in {build_dir}",
            file=sys.stderr,
        )
        print("hint: scripts/build-server.sh did not produce a package", file=sys.stderr)
        return 1

    try:
        deploy_package(remote_host, package_path)
    except subprocess.CalledProcessError as error:
        return error.returncode

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
