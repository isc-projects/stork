from datetime import datetime, timezone
import re
from time import time
from typing import Callable, List

import openapi_client
from openapi_client.api.users_api import UsersApi, User, Users, Groups, UserAccount
from openapi_client.api.services_api import ServicesApi, Machine, Machines, ConfigReports
from openapi_client.api.dhcp_api import DHCPApi, Subnets, Leases, Hosts, DhcpOverview
from openapi_client.api.events_api import EventsApi, Events
from openapi_client.model.event import Event

from core.compose import DockerCompose
from core.utils import NoSuccessException, wait_for_success
from core.wrappers.base import ComposeServiceWrapper
from core.log_parser import GoLogEntry, split_log_messages


class UnexpectedStatusCodeException(Exception):
    def __init__(self, expected, actual) -> None:
        super().__init__("Unexpected HTTP status. Expected: %d, got: %d"
                         % (expected, actual))


class UnexpectedEventException(Exception):
    def __init__(self, event):
        super().__init__("Unexpcted event occurs: %s" % event)


class Server(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str):
        super().__init__(compose, service_name)
        self._port = 8080
        self._address = self._compose.get_service_ip_address(
            self._service_name, "storknet"
        )
        url = "http://%s:%s/api" % (self._address, self._port)
        configuration = openapi_client.Configuration(
            host=url
        )
        self._api_client = openapi_client.ApiClient(configuration)

    @property
    def ip_address(self):
        return self._address

    # Lifetime

    def close(self):
        self._api_client.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()

    def _no_validate_kwargs(self, input=True, output=True):
        """
        Open API client arguments that disable the input and
        output validation for a specific call.

        Open API client for Python has built-in arguments that disable the
        validation: _check_input_type for input and _check_return_type for
        output. If the _check_input_type is False, the non-nullable check is
        performed instead of it. It recognizes that the _host_index (optional)
        parameter is None and raises the false alarm. To work around it, we
        need to specify this argument. But defining the host id suppresses
        using the URL from the configuration. Instead, the "/api" (from the
        Swagger file) is used. I didn't find an easy, legal way to inject the
        correct value. This hack overrides the method that produces the host
        list with a valid one.

        Returns
        -------
        dict
            Arguments that disable the input and output validation
        """
        params = {}
        if input:
            configuration = self._api_client.configuration
            configuration.get_host_settings = lambda: [
                {'url': configuration._base_path}
            ]
            params.update(_host_index=0, _check_input_type=False)
        if output:
            params.update(_check_return_type=False)
        return params

        # Authorize

    def log_in(self, username: str, password: str) -> User:
        api_instance = UsersApi(self._api_client)
        user, _, headers = api_instance.create_session(credentials=dict(
            useremail=username, userpassword=password
        ), _return_http_data_only=False)
        session_cookie = headers["Set-Cookie"]
        self._api_client.cookie = session_cookie
        return user

    def log_in_as_admin(self):
        return self.log_in("admin", "admin")

    # List / Search

    def list_users(self, limit=10, start=0) -> Users:
        api_instance = UsersApi(self._api_client)
        return api_instance.get_users(start=start, limit=limit)

    def list_groups(self, limit=10, start=0) -> Groups:
        api_instance = UsersApi(self._api_client)
        return api_instance.get_groups(start=start, limit=limit)

    def list_machines(self, authorized=None, limit=10, start=0) -> Machines:
        params = dict(start=start, limit=limit,
                      # This endpoint doesn't contain the applications.
                      **self._no_validate_kwargs())
        if authorized is not None:
            params["authorized"] = authorized
        api_instance = ServicesApi(self._api_client)
        return api_instance.get_machines(**params)

    def list_subnets(self, app_id=None, family: int = None, limit=10, start=0) -> Subnets:
        params = dict(start=start, limit=limit)
        if app_id is not None:
            params["appID"] = app_id
        if family is not None:
            params["dhcp_version"] = family

        api_instance = DHCPApi(self._api_client)
        return api_instance.get_subnets(**params)

    def list_events(self, daemon_type=None, app_type=None, machine_id=None,
                    user_id=None, limit=10, start=0) -> Events:
        params = dict(start=start, limit=limit)
        if daemon_type is not None:
            params["daemonType"] = daemon_type
        if app_type is not None:
            params["appType"] = app_type
        if machine_id is not None:
            params["machine"] = machine_id
        if user_id is not None:
            params["user"] = user_id

        api_instance = EventsApi(self._api_client)
        return api_instance.get_events(**params)

    def list_leases(self, text=None, host_id=None) -> Leases:
        params = {}
        if text != None:
            params["text"] = text
        if host_id != None:
            params["host_id"] = host_id

        api_instance = DHCPApi(self._api_client)
        return api_instance.get_leases(**params, **self._no_validate_kwargs())

    def list_hosts(self, text=None) -> Hosts:
        params = {}
        if text != None:
            params["text"] = text
        api_instance = DHCPApi(self._api_client)
        return api_instance.get_hosts(**params, **self._no_validate_kwargs())

    def list_config_reports(self, daemon_id: int, limit=10, start=0) -> ConfigReports:
        params = dict(limit=limit, start=start, id=daemon_id)
        api_instance = ServicesApi(self._api_client)
        return api_instance.get_daemon_config_reports(**params)

    def overview(self) -> DhcpOverview:
        api_instance = DHCPApi(self._api_client)
        return api_instance.get_dhcp_overview(**self._no_validate_kwargs())

        # Create

    def create_user(self, login, email, name, lastname, groups, password) -> User:
        user = User(id=0, login=login, email=email, name=name,
                    lastname=lastname, groups=groups)
        account = UserAccount(user, password)
        api_instance = UsersApi(self._api_client)
        return api_instance.create_user(account=account)

    # Read

    def read_machine_state(self, machine_id: int) -> Machine:
        api_instance = ServicesApi(self._api_client)
        return api_instance.get_machine_state(id=machine_id)

    # Update

    def update_machine(self, machine: Machine) -> Machine:
        api_instance = ServicesApi(self._api_client)

        return api_instance.update_machine(
            id=machine["id"],
            machine=machine,
            # This endpoint doesn't return the applications.
            **self._no_validate_kwargs()
        )

    # Complex

    def authorize_all_machines(self) -> Machines:
        machines = self.list_machines(False)
        machine: Machine
        for machine in machines["items"]:
            machine["authorized"] = True
            self.update_machine(machine)
        return machines

    # Waits

    @wait_for_success(UnexpectedStatusCodeException)
    def _wait_for_success_response(self, request, *args, **kwargs):
        return request(self, *args, **kwargs)

    def _wait_for_event(self,
                        expected_condition: Callable[[Event], bool] = None,
                        unexpected_condition: Callable[[
                            Event], bool] = None,
                        **kwargs):
        fetch_timestamp = datetime(1, 1, 1, tzinfo=timezone.utc)

        @wait_for_success(NoSuccessException)
        def worker():
            nonlocal fetch_timestamp

            events = self.list_events(limit=100, **kwargs)
            for event in reversed(events["items"]):
                # skip older events
                timestamp = event["created_at"]
                if timestamp < fetch_timestamp:
                    continue
                fetch_timestamp = timestamp

                if expected_condition is not None and expected_condition(event):
                    return
                if unexpected_condition is not None and unexpected_condition(event):
                    raise UnexpectedEventException(event)
            if expected_condition is not None:
                raise NoSuccessException()

        return worker()

    def _wait_for_logs(self, expected_condition: Callable[[GoLogEntry], bool] = None, start: datetime = None):
        if start is None:
            start = datetime.now(timezone.utc)

        @wait_for_success()
        def worker():
            stdout, _ = self._compose.get_logs(self._service_name)
            for log_entry in split_log_messages(stdout):
                if not log_entry.is_service(self._service_name):
                    continue
                if log_entry.timestamp < start:
                    continue
                go_entry = log_entry.as_go_safe()
                if go_entry is None:
                    continue
                if expected_condition(go_entry):
                    return go_entry
            raise NoSuccessException("missing expected log entry")
        return worker()

    def wait_for_next_machine_state(self, machine_id: int, start: datetime = None) -> Machine:
        if start is None:
            start = datetime.now(timezone.utc)

        @wait_for_success()
        def worker():
            state = self.read_machine_state(machine_id)
            last_visited = state["last_visited_at"]
            if last_visited < start:
                raise NoSuccessException()
            return state
        return worker()

    def wait_for_next_machine_states(self) -> List[Machine]:
        start = datetime.now(timezone.utc)
        machines = self.list_machines(authorized=True)
        states = []
        for machine in machines["items"]:
            state = self.wait_for_next_machine_state(
                machine["id"], start=start)
            states.append(state)
        return states

    _pattern_added_subnets = re.compile(
        r'added (?:(?:\d+ subnets)|(?:<subnet.*>)) to <daemon '
        r'id="(?P<daemon_id>\d+)" '
        r'name="(?P<daemon_name>.*)" '
        r'appId="(?P<app_id>\d+)"')

    def wait_for_adding_subnets(self, daemon_id: int = None, daemon_name: str = None, app_id: int = None):
        def condition(ev: Event):
            match = Server._pattern_added_subnets.search(ev["text"])
            if match is None:
                return False
            if daemon_id is not None and match.group("daemon_id") != str(daemon_id):
                return False
            if daemon_name is not None and match.group("daemon_name") != daemon_name:
                return False
            if app_id is not None and match.group("app_id") != str(app_id):
                return False
            return True

        self._wait_for_event(condition)

    def wait_for_failed_CA_communication(self, check_unauthorized=True):
        def condition(ev: Event):
            text = ev["text"]
            if not text.startswith("communication with CA daemon of"):
                return False
            if not text.endswith("failed"):
                return False

            if check_unauthorized and "Unauthorized" not in ev["details"]:
                return False
            return True
        self._wait_for_event(condition)

    def wait_for_statistics_pulling(self, app_type):
        expected = "completed pulling lease stats from %s apps" % app_type
        self._wait_for_logs(lambda ev: expected in ev.message)
