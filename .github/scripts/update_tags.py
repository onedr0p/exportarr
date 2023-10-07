import os
import sys

import re


FILES_WITH_IMAGE_REFS = [
    "./README.md",
    "./examples/compose/docker-compose.yaml",
    "./examples/kubernetes/lidarr-exporter.yaml",
    "./examples/kubernetes/radarr-exporter.yaml",
    "./examples/kubernetes/sonarr-exporter.yaml",
]

IMAGE_NAME = "ghcr.io/onedr0p/exportarr"
IMAGE_RE = re.compile(r'ghcr\.io/onedr0p/exportarr:v[0-9.]+', re.M)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("No tag passed to script! Exiting.")
        exit(1)

    tag = sys.argv[1]
    new_image = ":".join([IMAGE_NAME, tag])
    print(f"New Image Ref to write: {new_image}")

    for fi in FILES_WITH_IMAGE_REFS:
        print(f"Updating {fi}...")
        text = None
        with open(fi, "r") as f:
            text = f.read()

        text = IMAGE_RE.sub(new_image, text)
        with open(fi, "w") as f:
            f.write(text)
