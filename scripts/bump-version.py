#!/usr/bin/env python3
"""Synchronize the owner-authorized patch/minor version; never change major."""
import argparse
import json
import pathlib
import re

root = pathlib.Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('part', choices=['patch', 'minor'])
args = parser.parse_args()
version = (root / 'VERSION').read_text().strip()
if not re.fullmatch(r'\d+\.\d+\.\d+', version):
    raise SystemExit('VERSION must contain major.minor.patch')
major, minor, patch = map(int, version.split('.'))
if args.part == 'minor':
    minor, patch = minor + 1, 0
else:
    patch += 1
version = f'{major}.{minor}.{patch}'
(root / 'VERSION').write_text(version + '\n')
for name in ['package.json', 'package-lock.json']:
    path = root / 'frontend' / name
    data = json.loads(path.read_text())
    data['version'] = version
    if name == 'package-lock.json':
        data['packages']['']['version'] = version
    path.write_text(json.dumps(data, indent=2) + '\n')
print(version)
