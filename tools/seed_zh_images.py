# Seed the MarvelChampionsNext image cache with Chinese card images.
# by 兔兔 — 把懒人包里的简中卡图灌进 Go 项目的卡图缓存
#
# Usage:
#   python tools/seed_zh_images.py <pics_dir> [cache_dir]
#   pics_dir  = folder with images named by card code, e.g. 01001a.jpg / 40031.png
#   cache_dir = MarvelChampionsNext image cache root
#               (default: ./cache/images — matches cmd/server/main.go,
#                which uses NewImageCache(filepath.Join(MC_CACHE_DIR, "images")))
#
# The server (internal/api/images.go) stores cached images as {code}.img and
# records sha256[:16] per code in manifest.json. This script reproduces that
# layout exactly, so the server serves the Chinese images without ever
# hitting marvelcdb. Codes absent from the pics dir keep falling back to
# marvelcdb's English images.

import hashlib
import json
import re
import sys
from pathlib import Path

NAME_RE = re.compile(r"^([0-9]{5}[abc]?)(?:\.(?:jpg|jpeg|png|webp))$", re.IGNORECASE)


def valid_code(code: str) -> bool:
    # Mirror internal/api/images.go validCardCode: 5-6 chars of [0-9a-c].
    if not (5 <= len(code) <= 6):
        return False
    return all(ch in "0123456789abc" for ch in code)


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    src = Path(sys.argv[1])
    cache = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("cache") / "images"
    if not src.is_dir():
        print(f"ERR: source dir not found: {src}")
        return 1
    cache.mkdir(parents=True, exist_ok=True)

    manifest_path = cache / "manifest.json"
    manifest = {}
    if manifest_path.exists():
        try:
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        except Exception:
            manifest = {}

    written = unchanged = skipped = 0
    for f in sorted(src.iterdir()):
        if not f.is_file():
            continue
        m = NAME_RE.match(f.name)
        if not m or not valid_code(m.group(1)):
            skipped += 1
            continue
        code = m.group(1)
        body = f.read_bytes()
        digest = hashlib.sha256(body).hexdigest()[:16]
        target = cache / f"{code}.img"
        if manifest.get(code) == digest and target.exists():
            unchanged += 1
            continue
        target.write_bytes(body)
        manifest[code] = digest
        written += 1

    tmp = cache / "manifest.json.tmp"
    tmp.write_text(json.dumps(manifest, indent=1), encoding="utf-8")
    tmp.replace(manifest_path)
    print(f"done: manifest={len(manifest)} codes, written={written}, "
          f"unchanged={unchanged}, skipped={skipped}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
