# Legacy UI restoration

The owner's latest instruction supersedes the initial custom UI: the Option D application mirrors the legacy product's appearance and copy. The reference is `ssemakov/pass` at [`7cd8c6c0bc395fa3676c151a84c1657cda8cb481`](https://github.com/ssemakov/pass/tree/7cd8c6c0bc395fa3676c151a84c1657cda8cb481), together with the owner's login, home, results, hover and add-form screenshots. No real account records or screenshot images are checked into the fixtures.

## Matched behavior and styling

- **Window:** 904 × 534 logical pixels, matching `app/main.dev.js`. Wails' constructor uses content dimensions, so native startup explicitly applies the outer frame size before showing it. The native test reads the actual AppKit frame and requires exactly 904 × 534.
- **Theme:** legacy light theme, `#f4f4f5` background, white input/card surfaces, primary `#3f51b5`, original Roboto/Helvetica/Arial fallback stack, Material UI 4 spacing, typography, outlines, shadows and radii. OS dark mode does not introduce a new application palette.
- **Login:** one centered outlined password field, “Enter your Master Password”, with the original return-arrow icon; authentication failure uses “That didn't work”. No invented brand, headline, explanatory footer or onboarding copy.
- **Home:** 56-pixel plus button, “N secrets stored”, the original outlined “Search” field and an initially empty results area. Results appear after three characters and match name/URL. Only metadata is loaded for search.
- **Results:** full-width stacked white cards with bold names, `login`, `password`, `url`, the original five-asterisk mask and copy/reveal icons. Controls fade in on hover and remain keyboard reachable. A boolean password-presence flag reproduces legacy blank/null fields without prefetching secrets; embedded NUL is counted by byte length.
- **Add:** full-page dense outlined fields in the original order: “Item name”, “Username”, “Password”, “Website URL”; original “Cancel” and “Save” buttons and alignment. Edit reuses the form and keeps password replacement explicit. Delete uses the same Material icon and button language.
- **New actions:** native File menu items provide open/create/import, encrypted backup, password change and settings without adding a new toolbar to the legacy screens. New dialogs reuse the existing typography, outlined fields and buttons. Create retains the original two-step password/confirmation interaction and source placeholders.
- **Lock:** **⌘~**, replacing ⌘L, as the owner requested. The menu displays ⌘~; native bindings account for Wails naming the physical key as backquote with or without Shift. The native test sends actual AppKit key events and verifies ⌘L does not lock and ⌘~ does.

The UI keeps the existing storage, key derivation, transaction, migration, native authorization and session-fencing mechanisms. Native menu messages carry fixed action names, never paths or secrets. A pending open action is cancelled by a subsequent lock/transition instead of reopening a stale picker after an awaited lock completes.

## Reference and verification

The legacy `yarn.lock` resolves `@material-ui/core` **4.11.0**, icons **4.9.1** and React **16.13.1**. `scripts/render-legacy-ui.cjs` renders inert reference HTML with those exact direct versions and the original component props/theme. Generated references are committed in `frontend/tests/legacy-reference/`; old React/Material UI packages are isolated reference tooling and are not runtime dependencies of the new app. The app remains React 19/TypeScript/Vite with its Go/Wails backend.

`frontend/tests/legacy.spec.ts` compares login, home, results and add screens against these references at a 904 × 506 content viewport. It checks key bounding boxes within one logical pixel, exact computed font sizes/weights/colors, and rendered screenshots. Chromium is launched with LCD text antialiasing disabled so compositor-layer placement cannot switch only one page to colored subpixel text; this matches the legacy stylesheet’s intended grayscale smoothing. Pixel comparison excludes channel deltas of 32 or less to tolerate small rasterization differences; no more than 0.02% of pixels may exceed that tolerance. Local checks currently report zero pixels exceeding it for all four screens. The test also exercises hover opacity and the original “reveal password” tooltip.

The browser artifact contains synthetic screenshots of both reference and application screens and `legacy-comparison.json` with measured geometry/styles/differences. Functional tests cover CRUD, unchanged/empty passwords, backups, rotation/restore, historical migration, safe text rendering, stale reveal/status responses, keyboard focus and cancellation of a pending menu action. Native CI additionally verifies actual frame size, AppKit shortcuts, the production WebKit UI, clipboard operations, authorization/navigation denial and the existing storage/build gates on macOS 15/26 for both architectures.

Results and the tested commit will be recorded after the native rerun. Historical storage evidence in `docs/VALIDATION.md` and its manifest remains tied to its original tested commit.
