#!/bin/bash
set -e
mkdir -p build
cp server/pkg/PKGBUILD build/

cd build
makepkg -sf


