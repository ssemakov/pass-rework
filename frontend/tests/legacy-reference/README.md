# Inert legacy UI references

Generated with `scripts/render-legacy-ui.cjs` from the original component structures/theme at `ssemakov/pass@7cd8c6c0bc395fa3676c151a84c1657cda8cb481`. All records/passwords are public synthetic data. The HTML has no scripts or storage access.

Reference-only packages: React/React DOM 16.13.1, `@material-ui/core` 4.11.0 and `@material-ui/icons` 4.9.1, matching the legacy lockfile's UI versions. See the script header for the isolated install command. These packages are not added to the app's dependencies or production bundle.

Material UI and React are MIT licensed; the source repository's MIT license covers the original Pass component layout. Upstream notices: [Material UI v4 license](https://github.com/mui/material-ui/blob/v4.11.0/LICENSE), [React 16.12 license](https://github.com/facebook/react/blob/v16.13.1/LICENSE). Icon SVG paths used by the app are attributed in `frontend/src/legacy.tsx`.

The 2.0.2 comparison masks only the new `.home-lock` control using the exact page background. This is an explicit owner-requested addition to the historical header. Geometry/font/color checks remain unchanged for legacy elements; pixel tolerances are unchanged. Creation/import and new error messages follow the owner's updated requirements and have separate interaction/layout tests.
