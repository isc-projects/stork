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
import requests
import subprocess
import time
import traceback
from utils import setup_logger


logger = setup_logger(__name__)


class TimeoutException(Exception):
    pass


class NoSuchPortExposed(Exception):
    pass


class ContainerNotRunningException(Exception):
    def __init__(self, status):
        super().__init__("status=%s" % status)


class ContainerUnhealthyException(Exception):
    def __init__(self, status):
        super().__init__("status=%s" % status)


# Get a tuple of transient exceptions for which we'll retry. Other exceptions will be raised.
TRANSIENT_EXCEPTIONS = (TimeoutError, ConnectionError)
MAX_TRIES = int(os.environ.get("TC_MAX_TRIES", 120))
SLEEP_TIME = int(os.environ.get("TC_POOLING_INTERVAL", 1))


def wait_container_is_ready(*transient_exceptions):
    """
    Wait until container is ready.
    Function that spawn container should be decorated by this method
    Max wait is configured by config. Default is 120 sec.
    Polling interval is 1 sec.
    :return:
    """

    transient_exceptions = TRANSIENT_EXCEPTIONS + tuple(transient_exceptions)

    def wrapper(wrapped, instance, args, kwargs):
        exception = None
        logger.info("Waiting to be ready...")
        for _ in range(MAX_TRIES):
            try:
                return wrapped(*args, **kwargs)
            except transient_exceptions as e:
                logger.debug('container is not yet ready: %s', traceback.format_exc())
                time.sleep(SLEEP_TIME)
                exception = e
        raise TimeoutException(
            f'Wait time ({MAX_TRIES * SLEEP_TIME}s) exceeded for {wrapped.__name__}'
            f'(args: {args}, kwargs {kwargs}). Exception: {exception}'
        )

    return wrapper


class DockerCompose(object):
    """
    Manage docker compose environments.

    Parameters
    ----------
    filepath: str
        The relative directory containing the docker compose configuration file
    compose_file_name: str
        The file name of the docker compose configuration file
    pull: bool
        Attempts to pull images before launching environment
    build: bool
        Whether to build images referenced in the configuration file
    env_file: str
        Path to an env file containing environment variables to pass to docker compose

    Example
    -------
    ::

        with DockerCompose("/home/project",
                           compose_file_name=["docker-compose-1.yml", "docker-compose-2.yml"],
                           pull=True) as compose:
            host = compose.get_service_host("hub", 4444)
            port = compose.get_service_port("hub", 4444)
            driver = webdriver.Remote(
                command_executor=("http://{}:{}/wd/hub".format(host,port)),
                desired_capabilities=CHROME,
            )
            driver.get("http://automation-remarks.com")
            stdout, stderr = compose.get_logs()
            if stderr:
                print("Errors\\n:{}".format(stderr))


    .. code-block:: yaml

        hub:
        image: selenium/hub
        ports:
        - "4444:4444"
        firefox:
        image: selenium/node-firefox
        links:
            - hub
        expose:
            - "5555"
        chrome:
        image: selenium/node-chrome
        links:
            - hub
        expose:
            - "5555"
    """
    def __init__(
            self,
            filepath,
            compose_file_name="docker-compose.yml",
            pull=False,
            build=False,
            env_file=None,
            project_directory=".",
            service_names=None):
        self._filepath = filepath
        self._compose_file_names = compose_file_name if isinstance(
            compose_file_name, (list, tuple)
        ) else [compose_file_name]
        self._pull = pull
        self._build = build
        self._env_file = env_file
        self._project_directory = project_directory
        self._service_names = service_names if service_names is not None else []

    def __enter__(self):
        self.start(self._service_names)
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.stop()

    def docker_compose_command(self):
        """
        Returns command parts used for the docker compose commands

        Returns
        -------
        list[str]
            The docker compose command parts
        """
        docker_compose_cmd = ['docker-compose']
        for file in self._compose_file_names:
            docker_compose_cmd += ['-f', file]
        if self._env_file:
            docker_compose_cmd += ['--env-file', self._env_file]
        docker_compose_cmd += ["--project-directory", self._project_directory]
        return docker_compose_cmd

    def start(self, service_names=[]):
        """
        Starts the docker compose environment.
        """
        if self._pull:
            pull_cmd = self.docker_compose_command() + ['pull'] + service_names
            self._call_command(cmd=pull_cmd)

        up_cmd = self.docker_compose_command() + ['up', '-d'] + service_names
        if self._build:
            up_cmd.append('--build')

        self._call_command(cmd=up_cmd)

    def stop(self):
        """
        Stops the docker compose environment.
        """
        down_cmd = self.docker_compose_command() + ['down', '-v']
        self._call_command(cmd=down_cmd)

    def get_logs(self):
        """
        Returns all log output from stdout and stderr

        Returns
        -------
        tuple[bytes, bytes]
            stdout, stderr
        """
        logs_cmd = self.docker_compose_command() + ["logs"]
        result = subprocess.run(
            logs_cmd,
            cwd=self._filepath,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        return result.stdout, result.stderr

    def exec_in_container(self, service_name, command):
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
        exec_cmd = self.docker_compose_command() + ['exec', '-T', service_name] + command
        result = subprocess.run(
            exec_cmd,
            cwd=self._filepath,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        return result.stdout.decode("utf-8"), result.stderr.decode("utf-8"), result.returncode

    def get_service_port(self, service_name, port):
        """
        Returns the mapped port for one of the services.

        Parameters
        ----------
        service_name: str
            Name of the docker compose service
        port: int
            The internal port to get the mapping for

        Returns
        -------
        str:
            The mapped port on the host
        """
        return self._get_service_info(service_name, port)[1]

    def get_service_host(self, service_name, port):
        """
        Returns the host for one of the services.

        Parameters
        ----------
        service_name: str
            Name of the docker compose service
        port: int
            The internal port to get the host for

        Returns
        -------
        str:
            The hostname for the service
        """
        return self._get_service_info(service_name, port)[0]

    def _get_service_info(self, service, port):
        port_cmd = self.docker_compose_command() + ["port", service, str(port)]
        output = subprocess.check_output(port_cmd, cwd=self._filepath).decode("utf-8")
        result = str(output).rstrip().split(":")
        if len(result) == 1:
            raise NoSuchPortExposed("Port {} was not exposed for service {}"
                                    .format(port, service))
        return result

    def _call_command(self, cmd, filepath=None):
        if filepath is None:
            filepath = self._filepath
        subprocess.call(cmd, cwd=filepath)

    @wait_container_is_ready(requests.exceptions.ConnectionError)
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
        output = subprocess.check_output(cmd).decode("utf-8")
        container_id = str(output).rstrip()
        return container_id

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
        container_id = self.get_container_id(service_name)
        cmd = ["docker", "inspect", "--format",
            r"'{{ .State.Status  }};{{ if .State.Health }}{{ .State.Health.Status }}{{ end }}'",
            container_id
        ]
        status = subprocess.check_output(cmd).decode("utf-8")
        status, health = status.split(";")
        if health == "":
            health = None
        return status, health
        
    @wait_container_is_ready(ContainerNotRunningException)
    def wait_for_healthy(self, service_name):
        """
        Waits for a healthy status of a given service. This feature was
        introduced in docker-compose v2, but it isn't implemented for v1.
        Note that the image of the service should provide the HEALTHCHECK
        statement.

        Parameters
        ----------
        service_name: str
            Name of the service from the compose file
        """
        
        status, health = self.get_service_status(service_name)
        if status != "running":
            raise ContainerNotRunningException(status)
        if health is not None and health != "healthy":
            raise ContainerUnhealthyException(health)
