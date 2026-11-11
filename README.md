# PassV2

Another password manager.

A macOS rework of the original [Pass](https://github.com/ssemakov/pass), with the same minimal UI and a modern Go stack. See the [original README](https://github.com/ssemakov/pass/blob/main/README.md) and [original Pass demo](https://github.com/ssemakov/pass/blob/main/demo/pass_0_1_0_demo.gif) for its history and UI reference.

- One portable, encrypted storage file. No account or cloud service required.
- Search, add, edit and delete secrets. Reveal or copy credentials when needed.
- Lock with **⌘~**, the Lock button, or the idle timer. Lock also follows sleep and screen lock.
- Import original Pass storage into a fresh encrypted file, keeping the original untouched and verifying every entry.
- Save verified encrypted backups and change the master password.

## Binaries

PassV2 targets **macOS 15 and 26**, on Apple Silicon and Intel. Build the app from source using the instructions below. Packaging produces `build/Pass.app` and an architecture-specific ZIP.

See [build and distribution](docs/DISTRIBUTION.md) for signing and notarization, and the [validation record](docs/VALIDATION.md) for platform test results.

## Getting Started

### Installing

Building requires macOS, Xcode Command Line Tools, Go **1.27.3**, Node **24**, Python 3, Make and curl.

First, clone the repository:

```sh
git clone https://github.com/ssemakov/pass-rework.git
cd pass-rework
```

Install the dependencies:

```sh
./scripts/prepare-sqlcipher.sh
npm --prefix frontend ci --ignore-scripts --no-audit --no-fund
```

Build and open the app:

```sh
./scripts/package-macos.sh
open build/Pass.app
```

### Starting Development

After changing Go or frontend code, quit the app, rebuild with `./scripts/package-macos.sh` and reopen it. The native app embeds the frontend and uses the bundled SQLCipher implementation.

For a frontend-only development server:

```sh
npm --prefix frontend run dev
```

Use the packaged native app to exercise storage operations; the frontend preview has no storage bindings.

### Running the tests

Run all local checks with synthetic storage files:

```sh
./scripts/verify-local.sh
```

This runs Go race tests, Go vet, the production frontend build, unit tests and browser tests. On macOS it also runs native restart tests, packaging and artifact checks. Logs and the results manifest are written to `artifacts/`. Verification does not depend on GitHub Actions.

For individual test commands and recorded results, see [testing and validation](docs/VALIDATION.md).

## Using PassV2

On first launch, create storage or choose **Import Storage…** or **Open Storage…**. To import from the original Pass, close the original app, select its encrypted storage file and a fresh destination, then enter the original password and a new password with confirmation. The import screen reports **Imported N secrets** after verification. See [migration](docs/MIGRATION.md).

Later launches remember the storage path and ask for its master password. Settings are saved in `~/Library/Application Support/PassV2/settings.json`; passwords and keys are never saved there. Earlier app settings are not migrated, and existing storage files stay in place.

Use the plus button to add a secret. Search starts after three characters, as in the original Pass. Copy, reveal, edit and delete controls appear on hover or keyboard focus. The native File menu contains storage, import, backup, password change and settings actions. See the [complete app flows](docs/FLOWS.md).

## Built With

- [Go](https://go.dev/) — native application and storage logic.
- [Wails 3](https://wails.io/) — Go desktop framework using the system WebKit view on macOS.
- [React](https://react.dev/) and [TypeScript](https://www.typescriptlang.org/) — the interface, matching the original Pass UI.
- [Vite](https://vite.dev/) — frontend development and build tools.
- [SQLCipher](https://github.com/sqlcipher/sqlcipher) — encrypted SQLite storage and transactions.

## Documentation

- [Storage encryption evidence and threat model](docs/SECURITY.md)
- [Legacy UI matching](docs/UI-RESTORATION.md)
- [Implementation review](docs/IMPLEMENTATION-REVIEW.md) and [2.0.2 review](docs/UX-2.0.2-REVIEW.md)
- [2.0.2 changes and verification](docs/UX-2.0.2.md)

## Versioning

We use [SemVer](https://semver.org/). The application version is recorded in [VERSION](VERSION), which supplies the native About and bundle versions. Patch versions are for fixes; minor versions are for features. Major version changes require the owner's explicit instruction.

## License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.
