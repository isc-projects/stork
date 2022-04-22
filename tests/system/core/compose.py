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
from typing import List
import requests
import subprocess
from core.utils import setup_logger, memoize, wait_for_success


logger = setup_logger(__name__)


class NoSuchPortExposed(Exception):
    pass


class ContainerExitedException(Exception):
    pass


class ContainerNotRunningException(Exception):
    def __init__(self, status):
        super().__init__("status=%s" % status)


class ContainerUnhealthyException(Exception):
    def __init__(self, status):
        super().__init__("status=%s" % status)


_INSPECT_DELIMITER = ";"
_INSPECT_NONE_MARK = "<@NONE@>"


@memoize
def _construct_inspect_format(properties: tuple[str, ...]) -> str:
    """3
    Prepares the format string in Docker (Go Templates) format.
    The properties with question mark at the end are optional. It means
    that Docker inspect will not raise exception if they are missing.

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
    >>> _construct_inspect_format([".State.Status", ".State.Optional?.Status"])
    {{ .State.Status }};{{ if index .State "Optional" }}{{ .State.Optional.Status }}{{ else }}<@NONE@>{{ end }}
    """
    formats = []
    component_delimiter = "."
    for property in properties:
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

        format_property = "%s{{ %s }}%s" % (
            "".join(begins),
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
    project_dir: str
        The relative directory containing the docker compose configuration file
    compose_file_name: str
        The file name of the docker compose configuration file
    pull: bool
        Attempts to pull images before launching environment
    build: bool
        Whether to build images referenced in the configuration file
    env_file: str
        Path to an env file containing environment variables to pass to docker compose
    """

    def __init__(
            self,
            project_directory,
            compose_file_name="docker-compose.yml",
            pull=False,
            build=False,
            env_file=None,
            env_vars=None,
            project_name=None,
            use_build_kit=True):
        self._project_directory = project_directory
        self._compose_file_names = compose_file_name if isinstance(
            compose_file_name, (list, tuple)
        ) else [compose_file_name]
        self._pull = pull
        self._build = build
        self._env_file = env_file
        self._env_vars = env_vars
        self._use_build_kit = use_build_kit

        if project_name is None:
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
        docker_compose_cmd = ['docker-compose', '--no-ansi',
                              "--project-directory", self._project_directory,
                              "--project-name", self._project_name]
        for file in self._compose_file_names:
            docker_compose_cmd += ['-f', file]
        if self._env_file:
            docker_compose_cmd += ['--env-file', self._env_file]
        return docker_compose_cmd

    def build(self, *service_names):
        build_cmd = self.docker_compose_command() + \
            ['build', *service_names]

        env = None
        if self._use_build_kit:
            env = {
                "COMPOSE_DOCKER_CLI_BUILD": "1",
                "DOCKER_BUILDKIT": "1"
            }

        self._call_command(cmd=build_cmd, env_vars=env,
                           capture_output=False)

    def pull(self, *service_names):
        pull_cmd = self.docker_compose_command() + ['pull', *service_names]
        self._call_command(cmd=pull_cmd, capture_output=False)

    def up(self, *service_names):
        up_cmd = self.docker_compose_command() + ['up', '-d', *service_names]
        self._call_command(cmd=up_cmd, capture_output=False)

    def start(self, *service_names):
        """
        Starts the docker compose environment.
        """
        if self._pull:
            self.pull(*service_names)

        if self._build:
            logger.info("Begin build containers")
            self.build(*service_names)
            logger.info("End build containers")

        self.up(*service_names)

    def stop(self):
        """
        Stops the docker compose environment.
        """
        down_cmd = self.docker_compose_command() + ['down', '-v']
        self._call_command(cmd=down_cmd)

    def run(self, service_name: str, *args: str):
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
        self._call_command(cmd=run_cmd)

    def get_logs(self, service_name: str = None):
        """
        Returns all log output from stdout and stderr

        Parameters
        ----------
        service_name: str
            Name of the service. If None then all logs are fetched.

        Returns
        -------
        tuple[bytes, bytes]
            stdout, stderr
        """
        opts = ["logs", "--no-color", "-t"]
        if service_name is not None:
            opts.append(service_name)

        logs_cmd = self.docker_compose_command() + opts
        _, stdout, stderr = self._call_command(logs_cmd)
        return stdout, stderr

    def exec_in_container(self, service_name, command, check=True):
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
            exec_cmd,
            check=check
        )

    def get_service_ip_address(self, service_name, network_name):
        """
        Returns the assigned IP address for one of the services.

        Parameters
        ----------
        service_name: str
            Name of the docker compose service
        network_name: str
            Name of the network

        Returns
        -------
        str:
            The IP address for the service in a given network
        """
        prefixed_network_name = "%s_%s" % (self._project_name, network_name)
        return self.inspect(service_name,
                            ".NetworkSettings.Networks.%s.IPAddress"
                            % prefixed_network_name)[0]

    def get_service_mapped_host_and_port(self, service_name, port):
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
        _, stdout, _ = self._call_command(port_cmd)
        result = stdout.split(":")
        if len(result) == 1:
            raise NoSuchPortExposed("Port {} was not exposed for service {}"
                                    .format(port, service_name))
        return result

    def _call_command(self, cmd, check=True, capture_output=True, env_vars=None):
        env = os.environ.copy()
        if self._env_vars is not None:
            env.update(self._env_vars)
        if env_vars is not None:
            env.update(env_vars)
        env["PWD"] = self._project_directory

        result = subprocess.run(cmd, check=check, cwd=self._project_directory,
                                capture_output=capture_output, env=env)
        stdout = result.stdout
        stderr = result.stderr
        if capture_output:
            stdout = stdout.decode("utf-8").rstrip()
            stderr = stderr.decode("utf-8").rstrip()
            return result.returncode, stdout, stderr
        return result.returncode, None, None

    @wait_for_success(requests.exceptions.ConnectionError)
    def wait_for(self, url):
        """
        Waits for a response from a given URL. This is typically used to
        block until a service in the environment has started and is responding.
        Note that it does not assert any sort of return code, only check that
        the connection was successful.

        Parameters
        ----------
        url: str
            URL from one of the services in the environment to use to wait on
        """
        requests.get(url)
        return self

    def get_container_id(self, service_name):
        cmd = self.docker_compose_command() + ["ps", "-q", service_name]
        _, container_id, _ = self._call_command(cmd)
        if container_id == "":
            return None
        return container_id

    def inspect(self, service_name, *properties: str) -> list[str]:
        """
        Returns the low-level information on Docker containers.

        Parameters
        ----------
            service_name: str
                Name of the service
            properties: tuple[str]
                The properites to fetch as full path with the components
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
        _, stdout, _ = self._call_command(cmd)

        # Split the values and parse none's.
        return [i if i != _INSPECT_NONE_MARK else None
                for i in stdout.split(_INSPECT_DELIMITER)]

    def get_service_status(self, service_name):
        """
        Returns the container and health (if available) status for the service.

        Parameters
        ----------
        service_name: str
            Name of the service

        Returns
        -------
        tuple[str, str]
            container status, health status or None
        """
        status, health = self.inspect(service_name, ".State.Status",
                                      ".State.Health?.Status")
        return status, health

    def is_operational(self, service_name):
        """Return true if the service is in the running state and healthy
        (if the HEALTCHECK is specified)"""
        try:
            status, health = self.get_service_status(service_name)
        except LookupError:
            return False
        return status == "running" and (health is None or health == "healthy")

    @wait_for_success(ContainerNotRunningException)
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

        status, health = self.get_service_status(service_name)
        if status == "exited":
            # break
            raise ContainerExitedException()
        if status != "running":
            # continue
            raise ContainerNotRunningException(status)
        if health is not None:
            if health == "starting":
                # continue
                raise ContainerNotRunningException(health)
            if health == "unhealthy":
                # break
                raise ContainerUnhealthyException(health)
