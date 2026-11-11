# Project instructions

- Work on the owner's Option D branch; the owner chooses merges to main.
- Do not depend on or retry GitHub Actions. Run `./scripts/verify-local.sh` with synthetic data.
- Keep passwords, keys, real storage files and account screenshots out of commits.
- `VERSION` is the application version used by packaging/About. Keep frontend package versions in sync.
- The owner explicitly requested **2.0.2** for this batch. For later delivered fixes, increment the patch version. For new features, increment the minor version and reset patch to zero. Never change the major version unless the owner explicitly requests it.
- Use `python3 scripts/bump-version.py patch` or `minor`; do not bump once per internal implementation commit.
