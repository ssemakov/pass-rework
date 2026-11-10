#!/usr/bin/env python3
"""Generate the native About/bundle version from VERSION on any host."""
import json
import pathlib
import plistlib
import re
import sys

root = pathlib.Path(__file__).resolve().parents[1]
version = (root / 'VERSION').read_text().strip()
if not re.fullmatch(r'\d+\.\d+\.\d+', version):
    raise SystemExit('Invalid application version')
for name in ['package.json', 'package-lock.json']:
    data = json.loads((root / 'frontend' / name).read_text())
    if data['version'] != version:
        raise SystemExit(f'{name} version differs from VERSION')
    if name == 'package-lock.json' and data['packages']['']['version'] != version:
        raise SystemExit('Lockfile package version differs from VERSION')
plist = {'CFBundleExecutable':'Pass', 'CFBundleIdentifier':'io.ssemakov.pass-rework',
         'CFBundleName':'PassV2', 'CFBundleDisplayName':'PassV2',
         'CFBundlePackageType':'APPL', 'CFBundleShortVersionString':version,
         'CFBundleVersion':version, 'LSMinimumSystemVersion':'15.0',
         'NSHighResolutionCapable':True, 'NSRequiresAquaSystemAppearance':False}
pathlib.Path(sys.argv[1]).write_bytes(plistlib.dumps(plist))
