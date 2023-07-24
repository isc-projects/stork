import subprocess
from typing import List, Tuple, Union
from core.compose import DockerCompose


class Perfdhcp:
    """
    A wrapper for the docker-compose Perfdhcp service.
    This service container is available only when the service is executed.
    It isn't continuously running.

    The perfdhcp service is built on top of the Kea image. It swaps the
    entry point with the perfdhcp executable and doesn't use the supervisor
    daemon.

    I didn't know how exactly this wrapper would be used. I imagine that there
    will be different traffic generation scenarios. We shouldn't specify the
    perfhcp parameters inside the system tests. Initially, there are only
    simple traffic generation functions.
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

    def _run(self, parameters: List[str]):
        """Calls the perfdhcp application."""
        status, stdout, stderr = self._compose.run(
            self._service_name, *parameters, check=False
        )
        if status not in (0, 3):
            raise subprocess.CalledProcessError(status, "perfdhcp", stdout, stderr)
        if status == 3:
            print(stdout)

    @staticmethod
    def _generate_traffic_flags(
        family: int,
        target: Union[str, List[str]],
        mac_prefix: str = None,
        option: Tuple[str, str] = None,
        duid_prefix: str = None,
    ):
        """
        Generates the parameters set for perfdhcp. See perfdhcp documentation
        for details.
        Warning! There is a list of noticed issues with Perfdhcp:
            - duid value isn't recognized by the Kea class selector.
            - cannot use IPv6 addresses, Kea doesn't respond; it's related to
              Docker networking and unicast/multicast addresses on which Kea
              listens; a lot of time wasted here
        """

        flags = [
            # IP family
            f"-{family}",
            # Ratio
            "-r",
            "1",
            # Range
            "-R",
            "10",
            # Test period
            "-p",
            "10",
            # Exit wait time
            "-W",
            "10",
        ]

        if mac_prefix is not None:
            flags.append("-b")
            flags.append("mac=" + mac_prefix + ":00:00:00:00")
        if option is not None:
            flags.append(f"-o{option[0]},{option[1]}")
        if duid_prefix is not None:
            flags.append("-b")
            flags.append(f"duid={duid_prefix}00000000")

        if isinstance(target, str):
            flags.append(target)
        else:
            flags += target

        return flags

    def generate_ipv4_traffic(self, ip_address: str, mac_prefix=None, option=None):
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
        flags = Perfdhcp._generate_traffic_flags(
            family=4, target=ip_address, mac_prefix=mac_prefix, option=option
        )
        self._run(flags)

    def generate_ipv6_traffic(self, interface: str, option=None, duid_prefix=None):
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
        flags = Perfdhcp._generate_traffic_flags(
            family=6, target=["-l", interface], option=option, duid_prefix=duid_prefix
        )
        self._run(flags)
