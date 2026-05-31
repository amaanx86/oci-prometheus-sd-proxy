# Security

## Reporting Vulnerabilities

Do not open a GitHub issue to report security vulnerabilities.

If you believe you have found a security vulnerability, you can report it privately via either of these channels:

- **GitHub**: use [Report a vulnerability](https://github.com/amaanx86/oci-prometheus-sd-proxy/security/advisories/new) (preferred)
- **Email**: [amaanulhaq.s@outlook.com](mailto:amaanulhaq.s@outlook.com)

Include the following in your report:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Proof of concept (if available)

You will receive an acknowledgment within 48 hours. Once confirmed, critical vulnerabilities will be fixed within 90 days and all others within 180 days. You will be notified when a fix is released, and credited for the disclosure if desired.

Do not disclose the vulnerability publicly until a fix has been released.

## Commit Verification

All commits and tags from the maintainer are GPG-signed. You can verify them using the following key:

| Field | Value |
|---|---|
| Maintainer | @amaanx86 |
| Email | amaanulhaq.s@outlook.com |
| Key ID | `FC47DEE803DB612A` |
| Fingerprint | `9A1D B246 6C1A 9E94 F7DF 38C3 FC47 DEE8 03DB 612A` |
| Algorithm | RSA 4096 |
| Keyserver | https://keyserver.ubuntu.com/pks/lookup?search=amaanulhaq.s%40outlook.com&op=get |

> **Note:** GitHub may display the ID of a signing subkey (e.g. `2BB500FE5A696562`) rather than the primary key ID. Commits and tags may be signed by the documented primary OpenPGP key or by signing subkeys under that primary key. You can confirm a subkey belongs to the primary key by running `gpg --list-keys --with-subkey-fingerprints FC47DEE803DB612A`.

Fetch and verify:

```bash
gpg --keyserver keyserver.ubuntu.com --recv-keys FC47DEE803DB612A
gpg --verify <signed-file-or-tag>
```

## Secure Development Practices

- All changes are reviewed before merging
- Dependencies are regularly audited for known vulnerabilities
- All commits must be GPG-signed (`git commit -S`)
- No secrets, API keys, or credentials are committed to the repository
- Security-relevant functionality is covered by tests

## Security Updates

Security fixes are released as versioned updates. Users should:

1. Keep dependencies up to date
2. Monitor GitHub releases for security patches
3. Review the changelog for security-related changes

## Deployment Guidance

When deploying oci-prometheus-sd-proxy:

- Follow the principle of least privilege for OCI IAM policies
- Use environment variables for all sensitive configuration
- Mount OCI API keys as read-only volumes
- Rotate the `SERVER_TOKEN` bearer token periodically
- Run behind a reverse proxy with TLS in production

## Non-Vulnerability Suggestions

Security feature requests and improvement suggestions are welcome as GitHub issues.

## Questions

For security-related questions, contact [amaanulhaq.s@outlook.com](mailto:amaanulhaq.s@outlook.com) or reach out via [LinkedIn](https://www.linkedin.com/in/amaanulhaqsiddiqui/).
