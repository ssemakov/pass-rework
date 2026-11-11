# Legacy UI restoration adversarial review

Reviewer: separate agent `/root/implementation_adversary`, 2026-11-08 (America/New_York).

Scope: the legacy-style React/CSS/SVG rewrite, native menu commands, Command-tilde lock bindings, window geometry, and password-presence metadata. The original [13-finding implementation review](IMPLEMENTATION-REVIEW.md) remains preserved; this review addresses the subsequent UI work.

**Source review complete: no unresolved blocking code finding.** The new menu-ordering issue and two smaller observations below are corrected. Final reviewed source is `460666df2939a507a9e8591755f86e96dc6c8ff5`. Its local checks passed, but GitHub did not start runners for its final native matrix. Native key/window evidence below belongs to the explicitly identified successful predecessor, not to an unexecuted final-source run.

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

The visual comparison uses source-derived inert references rendered with the historical React/Material UI versions, isolated from the production dependency tree. Its four-view geometry/color/pixel assertions are useful Chromium evidence. They remain separate from real macOS WebKit key dispatch, native frame sizing and operating-system text rendering; the precise native evidence and final-source gap are stated below.

## Visual comparison normalization

The first `7ede560e` browser CI failed only the results-view pixel comparison while ten functional tests passed. The reviewer independently downloaded artifact `13647833749` and compared the actual/reference PNGs: 4,902 pixels exceeded the existing 32-channel-delta threshold, and 4,901 of them were grayscale actual text versus colored LCD-subpixel reference text. Geometry and computed styles matched; login/home comparisons had zero changed pixels.

Using `--disable-lcd-text` for the shared Chromium launcher normalizes this compositor-dependent rasterization for both pages. The geometry/style assertions and pixel tolerance are unchanged. This is a test-rendering normalization, not a relaxed match criterion or product-style substitution. The independent four-view comparison remains meaningful for the legacy layout and colors.

The final parity pass also checked source defaults for card clipping, ButtonBase/disabled controls, typography, readiness focus and new-entry search reset. Long-value coverage must distinguish the card's legacy `overflow:hidden` property from the legacy parent Grid's behavior: the historical Grid can itself expand. The new document's viewport-width bound is a separate robustness assertion, not a claim that the old application's document always stayed bounded.

## Verification and remaining platform gap

Final source `460666df2939a507a9e8591755f86e96dc6c8ff5` has recorded local passes for the production build, three Vitest tests, Go vet and all eleven browser integration tests in **21.9 s** using the same Playwright Chromium build 1194 as CI. Login, home, results and add views each recorded **zero changed pixels** above the unchanged threshold, alongside geometry/color/scroll assertions. The retained tests cover the menu-ordering cases, readiness focus, empty-field presence and long-value clipping. See the [evidence manifest](evidence/ui-2026-11-09.json).

The reviewer independently checked successful predecessor [run 42559048614](https://github.com/ssemakov/pass-rework/actions/runs/42559048614) at exact source `5a420d385d0fe752cb2d77ef1246d35a93c8d435`: all five jobs succeeded. Each native log contains **“Native window frame verified: 904x534”** and **“Native WebKit smoke passed: explicit UI assertions complete”**, plus the password-presence regression and CommonCrypto configuration pass. That smoke sends real AppKit `NSEvent` key events and requires Command-L to leave the vault unlocked and Command-tilde to lock it before reporting success.

| Platform | Independently inspected predecessor job |
|---|---|
| macOS 15 / Apple Silicon | [128971174720](https://github.com/ssemakov/pass-rework/actions/runs/42559048614/job/128971174720) |
| macOS 15 / Intel | [128971174781](https://github.com/ssemakov/pass-rework/actions/runs/42559048614/job/128971174781) |
| macOS 26 / Apple Silicon | [128971174962](https://github.com/ssemakov/pass-rework/actions/runs/42559048614/job/128971174962) |
| macOS 26 / Intel | [128971174788](https://github.com/ssemakov/pass-rework/actions/runs/42559048614/job/128971174788) |

Predecessor browser job [128971174514](https://github.com/ssemakov/pass-rework/actions/runs/42559048614/job/128971174514) independently confirms eleven passing tests and zero changed pixels for all four reference views. The final source then adds the reviewed control-default/clipping/focus refinements and native picker wording; predecessor results are not presented as a native pass for those later changes.

For the final source, [push run 42561162570](https://github.com/ssemakov/pass-rework/actions/runs/42561162570) attempts 1–4 and [PR run 42561168036](https://github.com/ssemakov/pass-rework/actions/runs/42561168036) attempt 2 failed during runner startup. The reviewer independently inspected the latest full job metadata: every job had **zero steps and no runner assignment**; available log requests returned `BlobNotFound`. No checkout, build, application or test step executed in these attempts. The accessible evidence does not establish the account/platform cause, and no billing diagnosis is asserted. These are not observed application-test failures, but they leave the exact final-source native matrix unvalidated. A native rerun is still needed when runner admission is available; no final-source native success is claimed.

No real vaults or passwords were used. No storage-encryption redesign, commissioned audit, notarization or future-beta claim is added by this UI review.
