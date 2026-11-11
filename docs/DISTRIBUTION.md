# Build and distribution

Use the exact Go patch in `go.mod`, Node 24, a supported Xcode/SDK and a macOS build host. `package-macos.sh` builds the current host architecture with minimum deployment target 15.0. CI builds native arm64 and x86_64 separately on macOS 15 and 26; the artifacts are architecture-specific, not universal bundles.

```sh
./scripts/prepare-sqlcipher.sh
npm --prefix frontend ci --ignore-scripts --no-audit --no-fund
./scripts/package-macos.sh
./scripts/check-artifact.sh build/Pass.app/Contents/MacOS/Pass
```

Outputs: `build/Pass.app` and `build/Pass-macos-<architecture>.zip`. Download/extract the inner ZIP from CI to preserve executable permissions and bundle metadata. CI's ad-hoc signature verifies bundle integrity but is not an Apple Developer ID identity or notarization. Do not label those artifacts as notarized public releases.

For an owner-authorized distribution build, supply an existing Developer ID signing identity and existing `notarytool` keychain profile. Secrets are never arguments in the repository or CI configuration:

```sh
export PASS_SIGN_IDENTITY='Developer ID Application: Your Name (TEAMID)'
./scripts/package-macos.sh
./scripts/check-artifact.sh build/Pass.app/Contents/MacOS/Pass
export PASS_NOTARY_PROFILE='your-existing-keychain-profile'
./scripts/notarize-macos.sh
```

The signing script enables the hardened runtime and secure timestamps for Developer ID. The notarization script refuses a bundle without a Developer ID authority, submits the ZIP, waits for Apple's result, staples/validates the ticket, checks Gatekeeper acceptance and rebuilds the final ZIP. Actual Developer ID signing/notarization requires the owner's Apple credentials and has not been performed by this implementation session. The owner decides merge, notarized release and publication; these scripts do not publish anything.

The artifact checker records SHA-256, Go module/build metadata, native linkage and symbols. It rejects release builds with debug, devtools, server, MCP, purego, e2e or native-smoke tags; requires the SQLCipher key API in the binary; rejects undefined `sqlite3_*` storage symbols; and verifies the bundle signature. WebKit's own transitive use of Apple's SQLite is allowed: the storage calls must resolve to the bundled SQLCipher implementation.

Update Go/Wails/frontend/SQLCipher pins intentionally. Regenerate and review source hashes when SQLCipher changes. Re-review the native WebKit delegate category against Wails changes to avoid selector collisions or changed injection semantics. Re-run the complete four-platform native matrix and encryption/migration/browser gates after updates. A Go or npm advisory scan does not scan SQLCipher's C code; review upstream SQLCipher and provider/OS advisories as well.
