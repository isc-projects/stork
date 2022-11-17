"""
Docker Compose Support
======================

Allows to spin up services configured via :code:`docker-compose.yml`.

File adopted from testcontainers-python (Apache 2.0 license) project.

See: https://github.com/testcontainers/testcontainers-python
See: https://raw.githubusercontent.com/testcontainers/testcontainers-python/master/testcontainers/compose.py
"""


#
#    Licensed under the Apache License, Version 2.0 (the "License"); you may
#    not use this file except in compliance with the License. You may obtain
#    a copy of the License at
#
#         http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
#    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
#    License for the specific language governing permissions and limitations
#    under the License.


import os
from typing import Dict, List, Tuple
import subprocess
import sys

import yaml

from core.utils import setup_logger, memoize, wait_for_success
from core.service_state import ServiceState


logger = setup_logger(__name__)


class NoSuchPortExposed(Exception):
    pass


class ContainerNotRunningException(Exception):
    def __init__(self, state):
        super().__init__(state)


class ContainerExitedException(Exception):
    def __init__(self, state: str):
        super().__init__(state)


_INSPECT_DELIMITER = ";"
_INSPECT_NONE_MARK = "<@NONE@>"


@memoize
def _construct_inspect_format(properties: Tuple[str, ...]) -> str:
    """
    Prepares the format string in Docker (Go Templates) format.
    The properties with question mark at the end are optional. It means
    that Docker inspect will not raise exception if they are missing.
    The property prepended with the "json" keyword will be serialized to JSON
    format.

    The constructed format string is cached to improve the performance. It
    causes that the properties container must be hashable.

    The property values will be delimited by the `_INSPECT_DELIMITER` delimiter.
    None values will be indicated by the `_INSPECT_NONE_MARK` special value.

    Parameters
    ----------
    properties : tuple[str]
        Paths to properties to fetch

    Returns
    -------
    str
        The Docker inspect format string

    Notes
    -----
    Thread safety: The function is pure. It has the same output for the same
    input. But access to the cache isn't synchronized (yet).
    Race may happen, but it shouldn't have any adverse effects.

    The cache isn't limited. We expect the function to be used with a small,
    fixed set of properties.

    This cache solution seems to be significant faster the `functools.lru_cache`.

    Examples
    --------
    >>> _construct_inspect_format([".State.Status", ".State.Optional?.Status", "json .State.Complex"])
    {{ .State.Status }};{{ if index .State "Optional" }}{{ .State.Optional.Status }}{{ else }}<@NONE@>{{ end }}{{ json .State.Complex }}
    """
    formats = []
    component_delimiter = "."
    json_prefix = "json "
    for property in properties:
        as_json = False
        if property.startswith(json_prefix):
            as_json = True
            property = property[len(json_prefix):]

        components = property.split(component_delimiter)
        begins = []
        path: List[str] = []
        for component in components:
            if component.endswith("?"):
                component = component[:-1]
                begins.append('{{ if index %s "%s" }}' % (
                    component_delimiter.join(path), component
                ))
            path.append(component)

        format_property = "%s{{ %s%s }}%s" % (
            "".join(begins),
            json_prefix if as_json else "",
            component_delimiter.join(path),
            "".join(["{{ else }}%s{{ end }}" %
                    _INSPECT_NONE_MARK, ] * len(begins))
        )
        formats.append(format_property)

    fmt = _INSPECT_DELIMITER.join(formats)
    return fmt


class DockerCompose(object):
    """
    Manage docker compose environments.

    Parameters
    ----------
    project_directory: str
        The relative directory containing the docker compose configuration file
    compose_file_name: str | list[str]
        The file name or list of the file names of the docker compose
        configuration file
    pull: bool
        Attempts to pull images before launching environment
    build: bool
        Whether to build images referenced in the configuration file
    env_file: str
        Path to an env file containing environment variables to pass to docker
        compose
    env_vars: dict
        The environment variables to pass to docker compose.
    project_name: str
        The docker compose project name. The current working directory is
        used by default.
    use_build_kit: bool
        Builds images using a BuiltKit mode
    default_mapped_address: str
        If provided, then the port command's default address (0.0.0.0) will be
        replaced with this value.
    """

    def __init__(
            self,
            project_directory: str,
            compose_file_name="docker-compose.yml",
            pull=False,
            build=False,
            env_file: str = None,
            env_vars: Dict[str, str] = None,
            build_args: Dict[str, str] = None,
            project_name: str = None,
            use_build_kit=True,
            default_mapped_hostname: str = None):
        self._project_directory = project_directory
        self._compose_file_names = compose_file_name if isinstance(
            compose_file_name, (list, tuple)
        ) else [compose_file_name]
        self._pull = pull
        self._build = build
        self._env_file = env_file
        self._env_vars = env_vars
        self._use_build_kit = use_build_kit
        self._default_mapped_hostname = default_mapped_hostname

        if build_args is not None:
            build_args_pairs = [("--build-arg", "%s=%s" % pair)
                                for pair in build_args.items()]
            # Flatten list
            build_args_strings = [item
                                  for pair in build_args_pairs
                                  for item in pair]

            self._build_args = build_args_strings
        else:
            self._build_args = []

        if project_name is None:
            # Mimics the docker-compose convention
            project_name = os.path.basename(os.path.abspath(project_directory))
        self._project_name = project_name

    def docker_compose_command(self):
        """
        Returns command parts used for the docker compose commands

        Returns
        -------
        list[str]
            The docker compose command parts
        """
        docker_compose_cmd = ['docker-compose', '--ansi', 'never',
                              "--project-directory", self._project_directory,
                              "--project-name", self._project_name]
        for file in self._compose_file_names:
            docker_compose_cmd += ['-f', file]
        if self._env_file:
            docker_compose_cmd += ['--env-file', self._env_file]
        return docker_compose_cmd

    def build(self, *service_names):
        """Builds the service containers. If no arguments are provided, it
        builds all containers. Supports BuildKit."""
        logger.info("Begin build containers")

        build_cmd = self.docker_compose_command() + [
            "build",
            *self._build_args,
            "--",
            *service_names
        ]

        env = None
        if self._use_build_kit:
            env = {
                "COMPOSE_DOCKER_CLI_BUILD": "1",
                "DOCKER_BUILDKIT": "1"
            }

        self._call_command(cmd=build_cmd, env_vars=env,
                           capture_output=False)
        logger.info("End build containers")

    def pull(self, *service_names):
        """Pull the images from a repository."""
        pull_cmd = self.docker_compose_command() + ['pull', *service_names]
        self._call_command(cmd=pull_cmd, capture_output=False)

    def up(self, *service_names):
        """Up the docker compose services."""
        up_cmd = self.docker_compose_command() + ['up', '-d', *service_names]
        self._call_command(cmd=up_cmd, capture_output=False)

    def start(self, *service_names):
        """
        Starts the docker compose environment.
        It can pull and build the containers if requested.
        """
        if self._pull:
            self.pull(*service_names)

        if self._build:
            self.build(*service_names)

        self.up(*service_names)

    def stop(self):
        """
        Stops the docker compose environment.
        """
        down_cmd = self.docker_compose_command() + ['down', '-v']
        self._call_command(cmd=down_cmd)

    def run(self, service_name: str, *args: str, check=True):
        """
        Run a one-off command on a service.

        Parameters
        ----------
        service_name : str
            Name of the service.
        """
        run_cmd = self.docker_compose_command() + [
            "run",
            "--no-deps",
            service_name,
            *args]
        return self._call_command(cmd=run_cmd, check=check)

    def logs(self, *service_names: str):
        """
        Returns all log output from stdout and stderr

        Parameters
        ----------
        service_names: str
            Names of the service. If empty then all logs are fetched.

        Returns
        -------
        tuple[bytes, bytes]
            stdout, stderr
        """
        opts = ["logs", "--no-color", "-t", *service_names]

        logs_cmd = self.docker_compose_command() + opts
        _, stdout, stderr = self._call_command(cmd=logs_cmd)
        return stdout, stderr

    def ps(self, *service_names: str):
        """
        Returns ps command output.

        Parameters
        ----------
        service_names: str
            Names of the service. If empty then all services are fetched.

        Returns
        -------
        str
            stdout
        """
        opts = ["ps", "--all", *service_names]

        ps_cmd = self.docker_compose_command() + opts
        _, stdout, _ = self._call_command(cmd=ps_cmd)
        return stdout

    def exec(self, service_name, command, check=True, capture_output=True):
        """
        Executes a command in the container of one of the services.

        Parameters
        ----------
        service_name: str
            Name of the docker compose service to run the command in
        command: list[str]
            The command to execute

        Returns
        -------
        tuple[str, str, int]
            stdout, stderr, return code
        """
        exec_cmd = self.docker_compose_command(
        ) + ['exec', '-T', service_name] + command
        return self._call_command(
            cmd=exec_cmd,
            check=check,
            capture_output=capture_output
        )

    def inspect(self, service_name, *properties: str) -> List[str]:
        """
        Returns the low-level information on Docker containers.

        Parameters
        ----------
            service_name: str
                Name of the service
            properties: tuple[str]
                The properties to fetch as full path with the components
                delimited by dot. If question mark at the end of the component
                marks it as optional

        Returns
        -------
        list[str | None]
            Containers values for each property
        """
        # Inspect isn't supported by the docker-compose.
        container_id = self.get_container_id(service_name)
        if container_id is None:
            raise LookupError(
                "container of the %s service not found" % service_name)

        format = _construct_inspect_format(properties)

        cmd = ["docker", "inspect", "--format", format, container_id]
        _, stdout, _ = self._call_command(cmd=cmd)

        # Split the values and parse none's.
        return [i if i != _INSPECT_NONE_MARK else None
                for i in stdout.split(_INSPECT_DELIMITER)]

    def inspect_raw(self, service_name) -> str:
        """Returns the low-level information on Docker containers as JSON."""
        container_id = self.get_container_id(service_name)
        if container_id is None:
            raise LookupError(
                "container of the %s service not found" % service_name)
        cmd = ["docker", "inspect", container_id]
        _, stdout, _ = self._call_command(cmd=cmd)
        return stdout

    def port(self, service_name, port) -> Tuple[str, int]:
        """
        Returns the mapped host and the mapped port for one of the services.

        Parameters
        ----------
        service_name: str
            Name of the docker compose service
        port: int
            The internal port to get the host for

        Returns
        -------
        tuple[str, str]:
            The hostname and port for the service
        """
        port_cmd = self.docker_compose_command() + ["port", service_name,
                                                    str(port)]
        _, stdout, _ = self._call_command(cmd=port_cmd)
        result = stdout.split(":")
        if len(result) == 1:
            raise NoSuchPortExposed("Port {} was not exposed for service {}"
                                    .format(port, service_name))
        mapped_host, mapped_port = result
        mapped_port = int(mapped_port)
        if self._default_mapped_hostname is not None and mapped_host == "0.0.0.0":
            mapped_host = self._default_mapped_hostname
        return mapped_host, mapped_port

    def get_service_ip_address(self, service_name, network_name, family):
        """
        Returns the assigned IP address for one of the services.
        It is an internal Docker IP address and it shouldn't be used
        to communicate with the service from the host.

        Parameters
        ----------
        service_name: str
            Name of the docker compose service
        network_name: str
            Name of the network
        family: int
            For family equals to 4 returns IPv4 address. Otherwise IPv6.

        Returns
        -------
        str:
            The IP address for the service in a given network
        """
        ip_property = "IPAddress" if family == 4 else "GlobalIPv6Address"
        prefixed_network_name = "%s_%s" % (self._project_name, network_name)
        return self.inspect(service_name,
                            ".NetworkSettings.Networks.%s.%s"
                            % (prefixed_network_name, ip_property))[0]

    def get_container_id(self, service_name):
        """
        Return a container ID assigned with a given service

        Parameters
        ----------
        service_name : str
            Name of the docker-compose service

        Returns
        -------
        str
            Container ID or None if any container doesn't exist

        Notes
        -----
        It doesn't support scaling.
        """
        cmd = self.docker_compose_command() + ["ps", "-q", service_name]
        _, container_id, _ = self._call_command(cmd=cmd)
        if container_id == "":
            return None
        return container_id

    def get_service_state(self, service_name) -> ServiceState:
        """
        Returns the container state (status and health (if available)) for the
        service.

        Parameters
        ----------
        service_name: str
            Name of the service

        Returns
        -------
        ServiceState
            container state
        """
        data = self.inspect(service_name,
                            ".State.Status",
                            ".State.ExitCode",
                            ".State.Health?.Status",
                            "json .State.Health?")
        return ServiceState(*data)

    def is_operational(self, service_name):
        """Return true if the service is in the running state and healthy
        (if the HEALTHCHECK is specified)"""
        try:
            state = self.get_service_state(service_name)
        except LookupError:
            return False
        return state.is_operational()

    def get_created_services(self):
        """Return the list of names of services that were created (includes
        operational and non-operational)"""
        opts = ["ps", "--services"]
        cmd = self.docker_compose_command() + opts

        _, stdout, _ = self._call_command(cmd)
        services = [line.strip() for line in stdout.split("\n")]
        created_services = []
        for service in services:
            container_id = self.get_container_id(service)
            if container_id is not None:
                created_services.append(service)
        return created_services

    @wait_for_success(ContainerNotRunningException,
                      wait_msg="Waiting to be operational...")
    def wait_for_operational(self, service_name):
        """
        Waits for the running and healthy (if the HEALTHCHECK is specified)
        status of a given service. This feature was introduced in
        docker-compose v2, but it isn't implemented for v1.

        Parameters
        ----------
        service_name: str
            Name of the service from the compose file
        """

        state = self.get_service_state(service_name)
        if state.is_exited():
            # Container cannot be recovered.
            raise ContainerExitedException(str(state))
        if not state.is_operational():
            raise ContainerNotRunningException(str(state))

    def get_pid(self, service_name, process_name):
        """
        Returns PID of the selected process belonging to the service.

        Parameters
        ----------
        service_name: str
            Name of the service
        process_name: str
            Name of the process which PID should be returned

        Returns
        -------
        int
            found PID or None
        """
        cmd = ["supervisorctl", "pid", process_name]
        _, stdout, _ = self.exec(service_name, cmd)

        try:
            return int(stdout)
        except Exception:
            return None

    def is_premium(self, service_name):
        """Checks if the given service is in the premium profile."""
        config = self._read_config_yaml()
        services_config = config['services']
        service_config = services_config.get(service_name)
        if service_config is None:
            raise LookupError(
                f'service {service_name} not found in the configuration')

        profiles = service_config.get("profiles")

        if profiles is None:
            # No profiles specified.
            return False

        return "premium" in profiles

    @memoize
    def _read_config_yaml(self):
        """Reads the configuration YAMS file and parses it."""
        config_cmd = self.docker_compose_command() + ["config", ]
        _, stdout, _ = self._call_command(cmd=config_cmd)
        return yaml.safe_load(stdout)

    def _call_command(self, cmd, check=True, capture_output=True, env_vars=None):
        env = os.environ.copy()
        if self._env_vars is not None:
            env.update(self._env_vars)
        if env_vars is not None:
            env.update(env_vars)
        env["PWD"] = self._project_directory

        opts = {}
        if sys.version_info >= (3, 7):
            opts["capture_output"] = capture_output
        elif capture_output:
            # Python 3.6 doesn't support capture output parameter
            opts["stdout"] = subprocess.PIPE
            opts["stderr"] = subprocess.PIPE

        result = subprocess.run(cmd, check=check, cwd=self._project_directory,
                                env=env, **opts)
        stdout: bytes = result.stdout
        stderr: bytes = result.stderr
        if capture_output:
            stdout: str = stdout.decode("utf-8").rstrip()
            stderr: str = stderr.decode("utf-8").rstrip()
            return result.returncode, stdout, stderr
        return result.returncode, None, None
