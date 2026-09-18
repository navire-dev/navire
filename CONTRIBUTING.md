# Contributing to Navire

Thank you for your interest in contributing to Navire.

Navire is open-source software distributed under the GNU Affero General Public License version 3 (AGPL-3.0).

Navire is currently preparing its first stable release. External contributions are not being accepted yet. The process described in this document will apply after the first stable release.

## After the first release

For significant changes, open an issue first so the proposal can be discussed before implementation begins. Small fixes and documentation improvements can usually be submitted directly as pull requests.

Please do not include credentials, private infrastructure details, or code that you are not authorized to contribute.

## Contribution terms

Navire uses the Developer Certificate of Origin (DCO) rather than a copyright assignment or a proprietary CLA.

By adding a `Signed-off-by` line to each commit, you certify that your contribution complies with the Developer Certificate of Origin. For details, see the [Developer Certificate of Origin](https://developercertificate.org/).

Example:

```text
Signed-off-by: Your Name <your.email@example.com>
```

> To add the `Signed-off-by` tag to your commit, use the `-s` option:
>
> `git commit -s -m "message"`

You retain the copyright in your contribution. The contribution may be used, modified, combined, and distributed as part of Navire under the AGPL-3.0.

Contributing to Navire does not transfer ownership of Navire, its source repositories, project name, trademarks, infrastructure, governance, or other contributors' work. It does not grant any contributor ownership or control of the project.

## Pull requests

1. Fork the repository and create a focused branch.
2. Keep changes small and scoped to one purpose when possible.
3. Format the code and add or update tests as appropriate.
4. Run the relevant project checks before opening the pull request.
5. Describe the behavior changed, the reason for the change, and any upgrade
   or compatibility impact.
6. Include a DCO sign-off on every commit.

Pull requests may be reviewed, revised, declined, or merged at the project's discretion. A contribution is not accepted until it has been merged by an authorized maintainer.

## Development checks

Use the repository's documented commands for formatting, tests, vulnerability scanning, and linting. Changes should follow idiomatic Go and preserve the separation of responsibilities described by the project architecture.

## Updating Dependencies

We use [Go Modules](https://github.com/golang/go/wiki/Modules) as the tool to manage vendor dependencies.

Use the following to update the version of all dependencies

```bash
$ go get -u
```

After the dependencies have been updated or added, you might run the following to cleanup the go module files:

```bash
$ go mod tidy
```

Please refer to [Go Modules](https://github.com/golang/go/wiki/Modules) for more details.

## Commit messages

Use clear Conventional Commit messages where practical:

```text
feat(events): add event validation
fix(metrics): remove stale dispatcher series
docs(roadmap): clarify the MVP scope
```

Breaking changes must use the ! notation:

```text
feat(api)!: version event ingestion
```

## Code of conduct

Please read and follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Reporting issues and security issues

Please report bugs and feature proposals through the project's [GitHub issue tracker](https://github.com/navire-dev/navire/issues).

Security issues should be reported privately according to [SECURITY.md](.github/SECURITY.md).

## Thank You

Thanks for your help! Navire would not be what it is today without your contributions.
