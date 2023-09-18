# Stork

<img align="right" src="/doc/static/stork-square-200px.png">

Stork is an open source ISC project providing a monitoring application and dashboard for
ISC Kea DHCP and (eventually) ISC BIND 9. A limited configuration management for Kea
is available and is expected to grow substantially in the near future.

The project is currently in rapid development, with bi-monthly releases rolling out new features.
See [wiki pages](https://gitlab.isc.org/isc-projects/stork/-/wikis/home) for useful
links to download page, release notes, self-guided demo, screenshots and much more.

For details, please see the [Stork Administrator Reference Manual](https://stork.readthedocs.io)
or the [Stork wiki](https://gitlab.isc.org/isc-projects/stork/-/wikis/home).

# Build Instructions

The easiest way to run Stork is to install it using
[RPM and deb packages](https://stork.readthedocs.io/en/latest/install.html#installing-from-packages).
The second easiest way is to use Docker (`rake demo:up` or `./stork-demo.sh`). However, it is
possible to run Stork without Docker. See the
[Installation section of the Stork ARM](https://stork.readthedocs.io/en/latest/install.html#installation).

# Reporting Issues

Please use the issue tracker on [ISC's GitLab](https://gitlab.isc.org/isc-projects/stork/-/issues)
to report issues and submit feature requests.

# Getting Involved

We have development releases every two months. If you'd like to get involved, feel free to subscribe to the
[stork-dev mailing list](https://lists.isc.org/mailman/listinfo/stork-dev) or look
at the [Stork project page](https://gitlab.isc.org/isc-projects/stork).
We're also on [GitHub](https://github.com/isc-projects/stork).

If you have a patch to send, by far the best way is to submit a
[merge request (MR) on GitLab](https://gitlab.isc.org/isc-projects/stork/-/merge_requests).
Stork developers use this system daily and you may expect a reasonably quick response.
The second alternative is to submit a [pull request (PR) on GitHub](https://github.com/isc-projects/stork/pulls).
This will also work, but this system is not monitored, so expect a delayed response.

# Screenshots

An example front page of the dashboard looks like this:
![Stork dashboard](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/22cf367aedaaad3ac8e42d066595dd7b/dashboard-1.1.png)

Many more Stork screenshots are available on the [Screenshots gallery](https://gitlab.isc.org/isc-projects/stork/-/wikis/Screenshots).

# Prometheus and Grafana

Stork provides support for statistics export in Prometheus format, which can then easily be shown in Grafana.

An example of Kea dashboard in Grafana, displaying data exported with Stork:
![grafana-kea4](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/97468f53d07c1b6eda7035c30fbd4de3/grafana-kea4.png)

BIND9 dashboard in Grafana, displaying data exported with Stork:
![grafana-bind2](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/6673c0a19962c535bf7e47d9fd0f46e5/grafana-bind2.png)
