# Stork

Stork is a new project led by ISC with the aim of delivering an ISC BIND 9 and ISC Kea DHCP use and monitoring dashboard. 
It is intended to be a spiritual successor of the earlier attempts - Kittiwake and Anterius.

It is currently in rapid development, with monthly releases with new features.

For details, please see [Stork Administrator Reference Manual](https://stork.readthedocs.io) or [Stork wiki](https://gitlab.isc.org/isc-projects/stork/-/wikis/home).

# Build instructions

The easiest way to run Stork is to install it using [RPM and deb packages](https://stork.readthedocs.io/en/v0.6.0/install.html#installing-from-packages).
The second easiest way is to use Docker (`rake docker_up`). However, it is
possible to run Stork without docker. See Installation section of the Stork ARM.

# Getting involved

Stork is in early stages of its development, but it's getting new features rapidly. We have
new release every month. If you'd like to get involved, feel free to subscribe to the
[stork-dev mailing list](https://lists.isc.org/mailman/listinfo/stork-dev) or look
at [Stork project page](https://gitlab.isc.org/isc-projects/stork). We're also on [github](https://github.com/isc-projects/stork).

# Screenshots

Here are some Stork screeshots.

Login screen - this is where it all starts.

![login](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/342aac544afeaa014bd4d52d328fe2f1/login.png)

Subnets list

![subnets](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/55770d48f64b4deb40341002de3cfd8e/subnets.png)

Networks list

![networks](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/743f066b5906c11f667674473c98b151/networks.png)

A dashboard!

![dashboard](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/64735611a93273cb6d5a2ece190d2755/dashboard.png)

Stork is able to monitor HA status and provides additional insight into failover events.

![ha-status](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/72010d2d5ad548bec65e4001108e172e/ha-status.png)

Stork is able to monitor BIND9 as well. You can have insight into how effective your caching is.

![bind9-details](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/30ba3ecf165d266be37146d9b0610927/bind9-details.png)

Stork can monitor multiple servers. Here's a list of servers (machines)

![machines-list](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/8636d5328a2b7d05f2eb6221485a67bf/machines-list.png)

There's a dedicated view for Kea processes (apps) running in your network

![kea-apps-list](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/15363e553cde30e8559c2a4a900f9d4d/kea-apps-list.png)

Stork provides support for Grafana. Here are some Kea and BIND9 dashboards:

![grafana-kea4](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/97468f53d07c1b6eda7035c30fbd4de3/grafana-kea4.png)

![grafana-bind](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/6a49fca880400b04ef2b84f196e4beaa/grafana-bind.png)

![grafana-bind2](https://gitlab.isc.org/isc-projects/stork/-/wikis/uploads/6673c0a19962c535bf7e47d9fd0f46e5/grafana-bind2.png)