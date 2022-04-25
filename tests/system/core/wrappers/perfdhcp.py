import subprocess
from typing import List, Tuple, Union
from core.compose import DockerCompose


class Perfdhcp:
    def __init__(self, compose: DockerCompose, service_name: str):
        self._compose = compose
        self._service_name = service_name

    def _call(self, parameters: List[str]):
        status, stdout, stderr = self._compose.run(
            self._service_name, *parameters, check=False)
        if status not in (0, 3):
            raise subprocess.CalledProcessError(
                status, "perfdhcp", stdout, stderr)

    def _generate_traffic_flags(self, family: int, target: Union[str, List[str]], mac_prefix: str = None,
                                option: Tuple[str, str] = None,
                                duid_prefix: str = None):
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
        flags = self._generate_traffic_flags(
            family=4,
            target=ip_address,
            mac_prefix=mac_prefix,
            option=option
        )
        self._call(flags)

    def generate_ipv6_traffic(self, interface: str, option=None,
                              duid_prefix=None):
        flags = self._generate_traffic_flags(
            family=6,
            target=["-l", interface],
            option=option,
            duid_prefix=duid_prefix)
        self._call(flags)
