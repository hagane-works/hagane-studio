# Contributing

Thank you for helping improve Hagane Works. Bug reports, documentation fixes, translations and focused
code changes are welcome.

## Before opening an issue

- Search existing issues first.
- Use the latest release or the current `main` branch when possible.
- Include the application version, Windows architecture and HOI4 version.
- Describe the smallest sequence that reproduces the problem.
- Remove personal paths, private mod content and game files from screenshots and logs.

Do not use a public issue for a security vulnerability. Follow [SECURITY.md](SECURITY.md) instead.

## Development setup

The project requires Go 1.25 or newer, Node.js 20 or newer and Git with submodule support.

```powershell
git clone --recurse-submodules https://github.com/hagane-works/hagane-studio.git
Set-Location hagane-studio
npm --prefix web ci
npm --prefix web run build
```

Run the backend and frontend development server as described in [README.md](README.md#development).

## Change guidelines

- Keep each pull request focused on one problem.
- Add or update tests for behavior changes.
- Do not commit game files, generated release archives, credentials or personal project data.
- Keep implementation code self-explanatory. Retain only comments required for licences, generated
  files or tool directives.
- Add user-facing text to all eight locale files under `web/messages/`.
- Preserve existing project and settings compatibility unless the change explicitly includes a
  migration.
- Document visible limitations instead of silently discarding unsupported data.

## Checks

Run these checks before submitting a pull request:

```powershell
go test -vet=off ./...
npm --prefix web run typecheck
npm --prefix web run lint
npm --prefix web test -- --run
npm --prefix web run build
```

If a change affects packaging, also run:

```powershell
./tools/release-windows.ps1 -Version local-check -SkipSourceArchive
```

## Licensing

By submitting a contribution, you agree that it may be distributed under this repository's
[GNU GPL v3.0-only licence](LICENSE). Only submit material that you have the right to license.
