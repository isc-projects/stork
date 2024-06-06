# Security Policy

## Supported Versions

Stork currently has no stable versions and all releases are considered development. Only the last development version is
supported:

| Version | Supported          | End-Of-Life           |
| ------- | ------------------ | --------------------- |
| 1.17.0  | :white_check_mark: | on release of 1.18.0  |
| 1.16.0  | :x:                | 2024 June 12          |
| 1.15.1  | :x:                | 2024 April 5          |
| earlier | :x:                |                       |

The first stable version will be 2.0.0. Once stable is released, the stable versions, denoted with even minor number,
will be supported for at least 6 months. The development versions will reach EOL as soon as next development or stable
version is released.

Limited past EOL support may be available to higher tier customers.
Please contact ISC sales, using this form: https://www.isc.org/contact/

## Reporting a Vulnerability

To report security vulnerability, please follow this instruction:

https://www.isc.org/reportbug/

Briefly, we prefer confidential issue on gitlab (not github). An issue is much better, because it's easier to get more
ISC engineers involved in it, evolve the case as more information is known, update or extra information, etc.

Second best is to send e-mail (possibly encrypted) to stork-security@isc.org.

## Software Defects and Security Vulnerability Disclosure Policy

ISC treats the security of its software products very seriously. This document discusses the evaluation of a defect
severity and the process in detail: https://kb.isc.org/docs/aa-00861

## Release Policy

Once the first stable (2.0.0) is published, we expect to have new stable versions (2.2.0, 2.4.0, ...) published
every six months or so. Once a new stable version is released, a new development cycle starts with monthly
development releases.

---

Stork team runs various security auditing tools. If a high severity issue is found in one of its dependencies, and the
underlying problem affects Stork, a release process is triggered that will lead to a Stork maintenance release.
If we would be unable to determine if Stork is affected, we will assume it is and will continue with the release.

For lower severity issues, the Stork team might choose to publish Operational notices that say that we are not
vulnerable to the vulnerability in one of our dependencies, explain that the vulnerability is minor or provide
workarounds how to mitigate.

The Stork team MUST NOT release a release with high or critical severity in any of its dependencies, if a fixed version
is available.

Different external dependencies treat severity differently. If CVSS score is published, we assume that CVSS >= 7.0
is considered high. If no CVSS score is published, the Stork team reserves the right to determine if the severity
is high or not.

The rules above apply to stable versions only. For development versions, it is uncommon to do a release out of the
ordinary release cycle.


## Further reading

The **Past advisories** for Stork can be found on the KB: https://kb.isc.org/docs
On the left hand panel, see the `Security Advisiories` in the `Kea DHCP` section.
