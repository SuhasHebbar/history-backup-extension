#!/bin/bash
resolutions=(128 48 32 16)
for size in "${resolutions[@]}"; do
    magick extension/icons/original.png -resize "$size"x"$size"! "extension/icons/icon-$size.png"
done