<!--
Copyright (C) 2024-2026 Internet Systems Consortium, Inc. ("ISC")

SPDX-License-Identifier: MPL-2.0

This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0.  If a copy of the MPL was not distributed with this
file, you can obtain one at https://mozilla.org/MPL/2.0/.

See the COPYRIGHT file distributed with this work for additional
information regarding copyright ownership.
-->
# Security Policy

ISC treats the security of its software products very seriously. ISC's Security Vulnerability Disclosure Policy
is documented in the relevant [ISC KnowledgeBase article](https://kb.isc.org/docs/aa-00861).

## Reporting Possible Software Vulnerabilities

To report a possible software vulnerability, please follow the instructions on [this page](https://www.isc.org/reportbug/).

We prefer that you [open a confidential issue in ISC's GitLab instance](https://gitlab.isc.org/isc-projects/stork/-/issues/new?issue[confidential]=true&issuable_template=Bug) (not GitHub). The GitLab issue creates a record,
is visible to all ISC engineers, and provides a shared communication channel with the reporter.

If it is not possible to create a GitLab issue, then send e-mail (encrypted if possible) to stork-security@isc.org. Do not
report any serious issues on the Stork project on GitHub, that is a mirror site and is not regularly monitored.

Please do not discuss undisclosed security vulnerabilities on any public mailing list. ISC has a long history of
handling reported vulnerabilities promptly and effectively and we respect and acknowledge responsible reporters.

If you have a crash, you may want to consult the KnowledgeBase article entitled ["What to do if your BIND, Kea DHCP,
Stork, or ISC DHCP server has crashed"](https://kb.isc.org/docs/aa-00340).

## Reporting Bugs and Lodging Feature Requests

Users are invited to visit our [Stork issues list in GitLab](https://gitlab.isc.org/isc-projects/stork/-/issues).
Before opening a new issue, please look and see if someone has already logged the bug you wish to report. You may
be able to add information to an existing report, or to find a workaround or updated status on the issue that
impacts you. We also track feature requests in the issue tracker, so please submit feature requests there as well.
Often it is helpful to first post these requirements on the [stork-users mailing list](https://lists.isc.org/mailman/listinfo/stork-users)
to clarify whether there is already a way to accomplish what you need.

Due to a large ticket backlog and an even larger quantity of incoming spam, we may sometimes be slow to respond,
especially if a bug is cosmetic or if a feature request is vague or low-priority. However, we truly appreciate and
depend on community-submitted bug reports and will address all reports of serious defects.

## No Bug Bounties

We are working with the interests of the greater Internet at heart, and we hope you are too. ISC does not
offer bug bounties. If you think you have found a bug in Stork, we encourage you to report it responsibly via a
confidential GitLab issue as described above; if verified, we will be happy to credit you in our Release Notes.

## Supported Versions

| Version | Kind        | Period                  | End-Of-Life                                     |
| ------- | ----------- | ----------------------- | ----------------------------------------------- |
| 2.5.x   | development | ~2 months               | August 2026 (on release of 2.6.0)               |
| 2.4.x   | stable      | ~6 months (+ ~3 months) | November 2026 (on release of 2.6.0 + 3 months)  |
| 2.3.x   | development | ~2 months               | February 2026                                   |
| 2.2.0   | stable      | ~6 months (+ ~3 months) | May 2026                                        |
| 2.1.x   | development | ~2 months               | June 2025                                       |
| 2.0.0   | stable      | ~6 months (+ ~3 months) | September 2025                                  |
| earlier | development | ~2 months               | November 2024                                   |

ISC's Software Support Policy and Version Numbering is explained in a [KnowledgeBase article](https://kb.isc.org/docs/aa-00896).
Limited past EOL support may be available to higher-tier customers. For more information, please contact ISC
sales at info@isc.org.

## Stork Security Advisories

**Past advisories** for Stork can be found in our [KnowledgeBase](https://kb.isc.org/).
On the left-hand panel, navigate to the Stork section and look for the `Security Advisories` folder.


