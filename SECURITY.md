# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.6.x   | ✅ Active support  |
| 0.5.x   | ✅ Active support  |
| < 0.5   | ❌ No patches      |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please email **ngockhoi96.dev@gmail.com** with the following details:

1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

You will receive a response within **48 hours** acknowledging receipt.
A fix will be developed privately and released as a patch version.

## Scope

In scope:

- Authentication bypass (API key or MCP auth)
- SQL injection
- Server-side request forgery (SSRF)
- Denial of service via resource exhaustion
- Information disclosure (API key leakage, etc.)
- Open redirect abuse
- MCP tool abuse (unauthorized operations via MCP endpoint)
- CORS misconfiguration on the public endpoint

Out of scope:

- Self-hosted instances with intentionally disabled auth
- Vulnerabilities in dependencies (report upstream)
- Social engineering
