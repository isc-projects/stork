import subprocess
from typing import List, Tuple, Union
from core.compose import DockerCompose


class Perfdhcp:
    """
    A wrapper for the docker-compose Perfdhcp service.
    This service container is available only when the service is executed.
    It isn't continuously running.

    It is a Kea service with a changed endpoint. It doesn't use the supervisor.

    I didn't know how exactly this wrapper would be used. I imagine that there
    will be different traffic generation scenarios. We shouldn't specify the
    perfhcp parameters inside the system tests. Initially, there are only
    simple generation functions.
    """

    def __init__(self, compose: DockerCompose, service_name: str):
        """
        A wrapper constructor.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the Perfdhcp docker-compose service.
        """
        self._compose = compose
        self._service_name = service_name

    def _call(self, parameters: List[str]):
        """Calls the perfdhcp application."""
        status, stdout, stderr = self._compose.run(
            self._service_name, *parameters, check=False)
        if status not in (0, 3):
            raise subprocess.CalledProcessError(
                status, "perfdhcp", stdout, stderr)

    def _generate_traffic_flags(self, family: int,
                                target: Union[str, List[str]],
                                mac_prefix: str = None,
                                option: Tuple[str, str] = None,
                                duid_prefix: str = None):
        """
        Generates the parameters set for perfdhcp. See perfdhcp documentation
        for details.
        Warning! Perfdhcp is a little buggy. List of notices issues:
            - duid value isn't recognized by the Kea class selector.
        """

        flags = [
            "-%d" % family,
            "-r", "1",
            "-R", "10",
            "-p", "10"
        ]

        if mac_prefix is not None:
            flags.append("-b")
            flags.append("mac=" + mac_prefix + ":00:00:00:00")
        if option is not None:
            flags.append("-o%s,%s" % option)
        if duid_prefix is not None:
            flags.append("-b")
            flags.append("duid=" + duid_prefix + "00000000")

        if type(target) == str:
            flags.append(target)
        else:
            flags += target

        return flags

    def generate_ipv4_traffic(self, ip_address: str, mac_prefix=None,
                              option=None):
        """
        Generate the IPv4 traffic.

        Parameters
        ----------
        ip_address : str
            The packets target. IPv4 address.
        mac_prefix : str, optional
            First 4 bytes of the MAC address in two groups separated by colon.
            E.g.: 12:34. If None then the default MAC is used.
        option : tuple, optional
            Two strings - key and value, by default None (not used)
        """
        flags = self._generate_traffic_flags(
            family=4,
            target=ip_address,
            mac_prefix=mac_prefix,
            option=option
        )
        self._call(flags)

    def generate_ipv6_traffic(self, interface: str, option=None,
                              duid_prefix=None):
        """
        Generate the IPv6 traffic.

        Parameters
        ----------
        interface : str
            The name of the interface, e.g.: eth0.
        option : _type_, optional
            Two strings - key and value, by default None (not used)
        duid_prefix : str, optional
            First 4 digits of DUID, by default None (not used)
        """
        flags = self._generate_traffic_flags(
            family=6,
            target=["-l", interface],
            option=option,
            duid_prefix=duid_prefix)
        self._call(flags)
