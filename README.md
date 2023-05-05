# console-github-auth

## How it works
- This component is deployed to cloud run (org: `nais.io`, project: `github-tenant-auth`), by `nais-terraform-modules`.
- It has credentials (app id / private key) for the nais/console github application.
- `nais-terraform-modules` grants invoke permissions to the console service account.
- When starting up, `console-github-auth` finds the appropriate installation id to create tokens for, based on the `GITHUB_ORG` env variable.
- When console needs to do requests targetting the GitHub API, it will:
	- use it's application default credentials to get an id token for it's own service account
	- use this to call `https://console-github-auth-something.run.app/createInstallationToken`
	- use the token returned by `console-github-auth` with the GitHub API. This token can only do operations targeting the specified GitHub organization.
