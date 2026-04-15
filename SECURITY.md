# Security Policy

## Supported Versions

We only support the latest version of Ground Control. If you find a security vulnerability, please ensure you are using the latest version before reporting.

| Version | Supported          |
| ------- | ------------------ |
| v0.x.x  | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you believe you have found a security vulnerability in Ground Control, please report it privately. We appreciate your help in keeping Ground Control secure.

### How to Report

Please send an email to the project maintainers (you can find contact details in the repository history or profile). 

Include the following in your report:
- A description of the vulnerability.
- Steps to reproduce the issue.
- Potential impact.
- Any suggested fixes.

### Our Commitment

We will:
- Acknowledge your report within 48 hours.
- Work with you to understand and resolve the issue.
- Provide a timeline for a fix.
- Give you credit for the discovery (if you wish).

## Security Best Practices for Users

- **Keep Ground Control updated**: Always use the latest stable release.
- **Isolate your NATS cluster**: Ensure your NATS server is properly secured with authentication and TLS.
- **Minimal privileges**: Run the Ground Control server and workers with the minimum privileges required.
