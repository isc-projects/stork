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
is documented in the relevant [ISC Knowledgebase article][1].

## Reporting possible security issues

To report a security vulnerability, please follow [this instruction][5].

Briefly, we prefer that you [open a confidential GitLab issue][2] (not Github). The GitLab issue creates a record,
is visible to all ISC engineers, and provides a shared communication channel with the reporter.

If it is not possible to create a GitLab issue, then send e-mail (encrypted if possible) to stork-security@isc.org.

Please do not discuss undisclosed security vulnerabilities on any public mailing list. ISC has a long history of
handling reported vulnerabilities promptly and effectively and we respect and acknowledge responsible reporters.

If you have a crash, you may want to consult the Knowledgebase article entitled ["What to do if your Stork has
crashed"][3].

## Supported Versions

Stork currently has no stable versions, and all releases are considered development ones. Only the last development
version is supported:

| Version        | Supported          | End-Of-Life             |
| -------------- | ------------------ | ----------------------- |
| latest 1.x.0   | :white_check_mark: | on release of 1.(x+1).0 |
| 1.16.0         | :x:                | 2024 June 12            |
| 1.15.1         | :x:                | 2024 April 5            |
| earlier        | :x:                |                         |

The first stable version will be 2.0.0. Stable versions, denoted with even minor numbers, will be supported for at least
6 months. Development versions will reach EOL as soon as the next development or stable version is released.

Limited past EOL support may be available to higher tier customers.
Please contact ISC sales, using the [contact form][4].

## Further reading

The **Past advisories** for Stork can be found on the KB: https://kb.isc.org/docs
On the left hand panel, see the `Security Advisiories` in the `Stork` section.

[1]: https://kb.isc.org/docs/aa-00861
[2]: https://gitlab.isc.org/isc-projects/stork/-/issues/new?issue[confidential]=true&issuable_template=Bug
[3]: https://kb.isc.org/docs/aa-00340
[4]: https://www.isc.org/contact/
[5]: https://www.isc.org/reportbug/
