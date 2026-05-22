# Security Policy

## Reporting

Report suspected vulnerabilities privately through GitHub Security Advisories for
this repository. If GHSA is unavailable to you, email security@openclaw.ai.

Do not open public issues for vulnerabilities or include OAuth client secrets,
refresh tokens, keyring data, emails, message contents, or exploit details in
public reports.

## Scope

In scope:

- OAuth, keyring, credential storage, and Gmail/Google API handling
- config/secrets loading and local filesystem boundaries
- command output that could disclose tokens, account data, or private mail data
- release workflows and package integrity

Out of scope:

- Google service outages, API changes, quotas, or account enforcement decisions
- compromise of a trusted local account, shell, filesystem, or device
- scanner-only findings without a reachable exploit path in supported usage

## Expectations

We prioritize reachable issues that affect credentials, private account data,
package integrity, or safe execution. Include the affected commit, platform,
minimal reproduction steps, and sanitized impact details.
