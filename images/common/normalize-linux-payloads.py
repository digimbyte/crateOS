from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
TARGETS = [
    REPO_ROOT / "images" / "common",
    REPO_ROOT / "images" / "x86" / "autoinstall",
    REPO_ROOT / "images" / "iso" / "autoinstall",
    REPO_ROOT / "images" / "iso" / "overlay",
    REPO_ROOT / "packaging" / "config",
    REPO_ROOT / "packaging" / "deb",
]


def main() -> None:
    for root in TARGETS:
        if not root.exists():
            continue
        for path in root.rglob("*"):
            if not path.is_file():
                continue
            try:
                data = path.read_bytes()
            except OSError:
                continue
            if b"\0" in data or b"\r" not in data:
                continue
            normalized = data.replace(b"\r\n", b"\n").replace(b"\r", b"\n")
            if normalized == data:
                continue
            path.write_bytes(normalized)


if __name__ == "__main__":
    main()
