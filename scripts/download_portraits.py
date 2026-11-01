#!/usr/bin/env python3
"""Fetch official ZZZ agent art for embedding in the Windows release.
Images are redistributed for user convenience; original artwork belongs to HoYoverse.
Fetch only the explicitly approved, pinned URLs, with bounded downloads.
"""
from pathlib import Path
import sys
import urllib.request
from io import BytesIO
from PIL import Image

SOURCES = {
    # Character artwork from https://zenless.hoyoverse.com/en-us/main?catchSpider=1
    "Severian": "https://fastcdn.hoyoverse.com/content-v2/nap/166076/77d73765548b8b853934bca30483bbf0_8151308603424226662.png",
    "Phoenix": "https://fastcdn.hoyoverse.com/content-v2/nap/166075/1aa5aba8052c4149131905385b053b6f_9200770839634652569.png",
    "Roxy": "https://fastcdn.hoyoverse.com/content-v2/nap/165591/1ecf0322900ace0d1bdeaaf059503791_2352214338355543397.png",
    "Claret": "https://fastcdn.hoyoverse.com/content-v2/nap/165587/618b974edc5bcde1764c0df7606c8213_483443791068031856.png",
    "Sigrid": "https://fastcdn.hoyoverse.com/content-v2/nap/164897/e0e291443b44acf30ad6434358fa7a06_517498375469235511.png",
    "Remielle": "https://fastcdn.hoyoverse.com/content-v2/nap/164866/1dfdd503fd531512cbd3ba6f35d681ec_7732394249204366837.png",
    # Original game asset portraits catalogued by Zenless Zone Zero Wiki.
    "Belle": "https://static.wikia.nocookie.net/zenless-zone-zero/images/6/6e/Agent_Belle_Portrait.png/revision/latest?cb=20240707002545",
    "Wise": "https://static.wikia.nocookie.net/zenless-zone-zero/images/e/e8/Agent_Wise_Portrait.png/revision/latest?cb=20240707002534",
}
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
DEST = Path(__file__).resolve().parents[1] / "assets" / "portraits"
MAX_BYTES = 12 * 1024 * 1024

def main():
    DEST.mkdir(parents=True, exist_ok=True)
    for name, url in SOURCES.items():
        req = urllib.request.Request(url, headers={
            "User-Agent": "Mozilla/5.0 (compatible; ZZZ-MANAGER/3.3.1; source: github.com/Gabriel-Liz2003/ZZZ-MANAGER)",
            "Accept": "image/png,image/*;q=0.9,*/*;q=0.5",
        })
        with urllib.request.urlopen(req, timeout=45) as resp:
            data = resp.read(MAX_BYTES + 1)
        if len(data) > MAX_BYTES or len(data) < 512:
            raise ValueError(f"{name}: image too large or empty, size={len(data)}")
        if not data.startswith(PNG_SIGNATURE):
            # Fandom may transparently serve WEBP for .png URLs. Normalize it
            # into PNG so the embedded server can always return image/png.
            try:
                with Image.open(BytesIO(data)) as img:
                    buf = BytesIO()
                    img.convert("RGBA").save(buf, format="PNG")
                    data = buf.getvalue()
            except Exception as exc:
                raise ValueError(f"{name}: unrecognized image encoding") from exc
        if not data.startswith(PNG_SIGNATURE) or len(data) > MAX_BYTES:
            raise ValueError(f"{name}: invalid normalized PNG ({len(data)} bytes)")
        path = DEST / (name + ".png")
        path.write_bytes(data)
        print(f"Embedded {name}: {len(data):,} bytes", flush=True)
    if len(list(DEST.glob("*.png"))) < len(SOURCES):
        raise RuntimeError("Missing some expected portrait files")
    print("All eight missing portraits prepared for embedding.", flush=True)

if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"Portrait download FAILED: {exc}", file=sys.stderr, flush=True)
        raise
