# Security

## Reporting Vulnerabilities

Do not open a GitHub issue to report security vulnerabilities.

If you believe you have found a security vulnerability, please report it directly to:

**Email**: [amaanulhaq.s@outlook.com](mailto:amaanulhaq.s@outlook.com)

Include the following in your report:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Proof of concept (if available)

You will receive an acknowledgment within 48 hours. Once the issue is confirmed, a timeline for resolution will be provided. You will be notified when a fix is released, and credited for the disclosure if desired.

Do not disclose the vulnerability publicly until a fix has been released.

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
