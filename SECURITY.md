<!--
Copyright (C) Internet Systems Consortium, Inc. ("ISC")

SPDX-License-Identifier: MPL-2.0

This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0.  If a copy of the MPL was not distributed with this
file, you can obtain one at https://mozilla.org/MPL/2.0/.

See the COPYRIGHT file distributed with this work for additional
information regarding copyright ownership.
-->
# Security Policy

ISC treats the security of its software products very seriously. ISC's Security Vulnerability Disclosure Policy
is documented in the relevant [ISC KnowledgeBase article][1].

For official ISC security policy, see [this KB article](https://kb.isc.org/docs/aa-00861). As a convenience to the reader,
below are the major points from the policy.

## Reporting possible security issues

To report a security vulnerability, please follow [this instruction][5].

Briefly, we prefer that you [open a confidential GitLab issue][2] (not Github). The GitLab issue creates a record,
is visible to all ISC engineers, and provides a shared communication channel with the reporter.

If it is not possible to create a GitLab issue, then send e-mail (encrypted if possible) to stork-security@isc.org.

Please do not discuss undisclosed security vulnerabilities on any public mailing list. ISC has a long history of
handling reported vulnerabilities promptly and effectively and we respect and acknowledge responsible reporters.

If you have a crash, you may want to consult the KnowledgeBase article entitled ["What to do if your Stork has
crashed"][3].

## Supported Versions

The first stable version is 2.0.0. Stable versions, denoted with even minor numbers, will be supported for at least 6
months plus 3 months of transition when we can provide critical updates. Development versions will reach EOL as soon as
the next development or stable version is released.

| Version | Kind        | Period                  | End-Of-Life                      |
| ------- | ----------- | ----------------------- | -------------------------------- |
| 2.2.0   | stable      | ~6 months (+ ~3 months) | on release of 2.4.0 + 3 months   |
| 2.1.x   | development | ~2 months               | on release of 2.1.(x+1) or 2.2.0 |
| 2.0.0   | stable      | ~6 months (+ ~3 months) | on release of 2.2.0 + 3 months   |
| earlier | development |                         | on release of 2.0.0              |

Limited past EOL support may be available to higher tier customers.
Please contact ISC sales, using the [contact form][4].

The Stork team may release a security release when a severe vulnerability is found. The vulnerability must have a high
CVSS score and affect any Stork component directly or allow an attack Kea or BIND 9 through Stork. We don't make a
security release if the vulnerability affects a third-party dependency in part Stork does not use.

If the Stork team recognizes a serious security issue, we will immediately notify higher-tier customers via internal
security channels. When the fix is ready, we will preannounce the release date (without technical details) on our
mailing list.

## Further reading

The **Past advisories** for Stork can be found on the [KnowledgeBase][6].
On the left hand panel, see the `Security Advisories` in the `Stork` section.

[1]: https://kb.isc.org/docs/aa-00861
[2]: https://gitlab.isc.org/isc-projects/stork/-/issues/new?issue[confidential]=true&issuable_template=Bug
[3]: https://kb.isc.org/docs/aa-00340
[4]: https://www.isc.org/contact/
[5]: https://www.isc.org/reportbug/
[6]: https://kb.isc.org/docs
