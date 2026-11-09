# Legacy UI restoration adversarial review

Reviewer: separate agent `/root/implementation_adversary`, 2026-11-08 (America/New_York).

Scope: the legacy-style React/CSS/SVG rewrite, native menu commands, Command-tilde lock bindings, window geometry, and password-presence metadata. The original [13-finding implementation review](IMPLEMENTATION-REVIEW.md) remains preserved; this review addresses the subsequent UI work.

**Source review complete: no unresolved blocking code finding.** The new menu-ordering issue and two smaller observations below are corrected. Actual macOS key dispatch/window-size evidence is pending the new native CI run; the earlier implementation's matrix does not establish the rewritten UI's behavior.

## Findings

### UI-01 — Medium / resolved: a superseded menu transition can open a late dialog

The initial `begin()` implementation awaits `lock()` and then unconditionally opens its requested panel and potentially the native file picker. A second Lock or menu transition while the first acknowledgement is delayed can supersede that request, but the first continuation still resumes afterward. It can open a stale Open/Create/Import flow after the user's later lock intent. This path does not restore old credentials; it is an asynchronous command/cancellation defect.

Correction reviewed: local locks share one closing promise and keep controls busy through the Lock/State handshake. Menu transitions wait for that promise and use a nonce so only the latest intent may open a panel/picker. The closing continuation no longer calls a late `updateState()`/scrub. Matching owned lock notifications preserve handshake busy state and are ignored after their locked generation is already recorded.

The handshake also captures the exact expected generation. An additional native lock invalidates queued intent when State exceeds that generation, even if the corresponding event has not yet arrived. This closes an event-ordering gap which a DOM-only second-lock test would miss.

Retained browser regressions cover delayed acknowledgement followed by a second Command-tilde, a newer Create request waiting for the same handshake and preserving its draft after a late owned event, and a direct backend/native extra lock with its notification withheld until after State. No previous credentials are restored by these paths.

### UI-02 — Low / resolved: shortcut documentation still names Command-L

The README's feature list initially advertised `⌘L`, while the new native and frontend code use Command-tilde/backquote. The current instructions now say `⌘~`; the earlier implementation review remains intact as history.

### UI-03 — Low / resolved: the new smoke-only key route skipped the harness guard

The initial `/__smoke_key` branch ran before the smoke handler's POST/capability/native-window-ID check. It is compiled only into the synthetic native test binary and was not a production route. The shared guard now precedes both key and result dispatch, preserving the harness authorization contract.

## Source assessment

- The metadata list still selects `NULL` in the password-value slot. Its new boolean `HasPassword` reflects nonempty text using BLOB length, preserving leading-NUL semantics without prefetching password values. The schema and cipher format are unchanged. The independent `TestMetadataPasswordPresenceWithoutSecretPrefetch` race run passed in **1.348 s** for NULL, empty and leading-NUL values.
- Reveal delivery retains the session fence and a nonce invalidated by search/concealment/lock. Cards render secrets only for the selected matching entry. React text rendering and static SVG paths do not introduce an HTML/URL injection sink.
- Native menu dispatch uses fixed allowlisted command names encoded with JSON. It carries no paths, keys or record values and continues through the existing authorized facade and native selection grants. The frontend ignores unknown commands.
- The pinned Wails menu implementation runs click callbacks on goroutines. The native menu's direct manager lock therefore does not wait on storage from the AppKit main thread; the previous shutdown/clipboard correction remains intact.
- Command-L was removed from the explicit application/frontend lock bindings. Both shifted and unshifted physical backquote bindings target the manager; the menu displays the requested Command-tilde. Browser key events verify frontend behavior, while the native smoke must verify real macOS dispatch and the 904×534 outer frame.
- The initial locked-State exception avoids clearing a form before any active generation exists. Successful authentication, stale responses and old lock notifications continue to use generation/fence checks. The corrected menu continuation now applies that ordering discipline to panel and picker side effects as well.

The visual comparison uses source-derived inert references rendered with the historical React/Material UI versions, isolated from the production dependency tree. Its four-view geometry/color/pixel assertions are useful Chromium evidence. They are separate from real macOS WebKit key dispatch, native frame sizing and operating-system text rendering; the new native run must provide that evidence before those results are claimed.

## Visual comparison normalization

The first `7ede560e` browser CI failed only the results-view pixel comparison while ten functional tests passed. The reviewer independently downloaded artifact `13647833749` and compared the actual/reference PNGs: 4,902 pixels exceeded the existing 32-channel-delta threshold, and 4,901 of them were grayscale actual text versus colored LCD-subpixel reference text. Geometry and computed styles matched; login/home comparisons had zero changed pixels.

Using `--disable-lcd-text` for the shared Chromium launcher normalizes this compositor-dependent rasterization for both pages. The geometry/style assertions and pixel tolerance are unchanged. This is a test-rendering normalization, not a relaxed match criterion or product-style substitution. The independent four-view comparison remains meaningful for the legacy layout and colors.

The final parity pass also checked source defaults for card clipping, ButtonBase/disabled controls, typography, readiness focus and new-entry search reset. Long-value coverage must distinguish the card's legacy `overflow:hidden` property from the legacy parent Grid's behavior: the historical Grid can itself expand. The new document's viewport-width bound is a separate robustness assertion, not a claim that the old application's document always stayed bounded.

No real vaults or passwords were used. No storage-encryption redesign, commissioned audit, notarization or future-beta claim is added by this UI review.
