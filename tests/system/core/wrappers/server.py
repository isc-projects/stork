from datetime import datetime, timezone
import re
from time import time
from typing import Callable, List, Optional
from contextlib import contextmanager

import openapi_client
from openapi_client.api.users_api import UsersApi, User, Users, Groups, UserAccount
from openapi_client.api.services_api import ServicesApi, Machine, Machines, ConfigReports
from openapi_client.api.dhcp_api import DHCPApi, Subnets, Leases, Hosts, DhcpOverview
from openapi_client.api.events_api import EventsApi, Events
from openapi_client.api.general_api import GeneralApi, Version
from openapi_client.model.event import Event
from openapi_client.model.subnet import Subnet
from openapi_client.model.local_subnet import LocalSubnet

from core.compose import DockerCompose
from core.utils import NoSuccessException, wait_for_success
from core.wrappers.base import ComposeServiceWrapper


class Server(ComposeServiceWrapper):
    """
    A wrapper for the docker-compose service containing Stork Server.
    """

    def __init__(self, compose: DockerCompose, service_name: str):
        """
        A wrapper constructor.

        It assumes that the server is available on port 8080 and it's
        connected to the storknet network.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service
        """
        super().__init__(compose, service_name)
        internal_port = 8080
        mapped = self._compose.port(service_name, internal_port)
        url = "http://%s:%d/api" % mapped
        configuration = openapi_client.Configuration(host=url)
        self._api_client = openapi_client.ApiClient(configuration)

    def close(self):
        """Free the resources used by the wrapper."""
        self._api_client.close()

    def __enter__(self):
        """
        Context manager entry point. It does nothing but the language requires
        it.
        """
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        """
        Free the resources used by the wrapper on leave the context bounds.
        """
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

        Parameters
        ----------
        input : bool, optional
            Suppress the input types validation, by default True
        output : bool, optional
            Suppress the output types validation, by default True

        Returns
        -------
        dict
            Parameters that disable the input and output validation
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

    @staticmethod
    def _parse_date(date):
        """
        Parses the GO timestamp if it is a string otherwise, lefts it as is.
        """
        if type(date) == str:
            date = datetime.strptime(date, "%Y-%m-%dT%H:%M:%S.%fZ")
            date = date.replace(tzinfo=timezone.utc)
        return date

    @staticmethod
    def _is_before(first_date, second_date):
        """
        Checks if the first date is before the second. The dates can be Go
        timestamps or Python datetime objects.
        """
        first_date = Server._parse_date(first_date)
        second_date = Server._parse_date(second_date)
        return first_date < second_date

    # Authentication

    def log_in(self, username: str, password: str) -> User:
        """Logs in a user. Returns the user info."""
        api_instance = UsersApi(self._api_client)
        user, _, headers = api_instance.create_session(credentials=dict(
            useremail=username, userpassword=password
        ), _return_http_data_only=False)
        session_cookie = headers["Set-Cookie"]
        self._api_client.cookie = session_cookie
        return user

    def log_in_as_admin(self):
        """Logs in an admin. Returns the user info."""
        return self.log_in("admin", "admin")

    # List / Search

    def list_users(self, limit=10, start=0) -> Users:
        """Lists the users."""
        api_instance = UsersApi(self._api_client)
        return api_instance.get_users(start=start, limit=limit)

    def list_groups(self, limit=10, start=0) -> Groups:
        """Lists the groups."""
        api_instance = UsersApi(self._api_client)
        return api_instance.get_groups(start=start, limit=limit)

    def list_machines(self, authorized=None, limit=10, start=0) -> Machines:
        """Lists the machines."""
        params = dict(start=start, limit=limit)
        if authorized is not None:
            params["authorized"] = authorized
        api_instance = ServicesApi(self._api_client)
        return api_instance.get_machines(**params)

    def list_subnets(self, app_id=None, family: int = None,
                     limit=10, start=0) -> Subnets:
        """Lists the subnets from a given application and/or family."""
        params = dict(start=start, limit=limit)
        if app_id is not None:
            params["app_id"] = app_id
        if family is not None:
            params["dhcp_version"] = family

        api_instance = DHCPApi(self._api_client)
        return api_instance.get_subnets(**params)

    def list_events(self, daemon_type=None, app_type=None, machine_id=None,
                    user_id=None, limit=10, start=0) -> Events:
        """
        Lists the events.

        Parameters
        ----------
        daemon_type : str, optional
            Daemon type, e.g. 'named', 'dhcp4', 'dhcp6', 'ca', by default None
        app_type : str, optional
            App type, e.g. 'kea' or 'bind9', by default None
        machine_id : int, optional
            Machine ID, by default None
        user_id : int, optional
            User ID, by default None
        limit : int, optional
            Maximal number of entries, by default 10
        start : int, optional
            List offset, by default 0

        Returns
        -------
        Events
            List of events
        """
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
        """
        Lists the leases

        Parameters
        ----------
        text : str, optional
            Should contain an IP address, MAC address, client id or hostname.
            It is mutually exclusive with the hostId parameter, by default None
        host_id : int, optional
            Identifier of the host for which leases should be searched. It is
            mutually exclusive with the text parameter, by default None

        Returns
        -------
        Leases
            List of leases
        """
        params = {}
        if text != None:
            params["text"] = text
        if host_id != None:
            params["host_id"] = host_id

        api_instance = DHCPApi(self._api_client)
        return api_instance.get_leases(**params, **self._no_validate_kwargs())

    def list_hosts(self, text=None) -> Hosts:
        """Lists the hosts based on the host identifier."""
        params = {}
        if text != None:
            params["text"] = text
        api_instance = DHCPApi(self._api_client)
        return api_instance.get_hosts(**params, **self._no_validate_kwargs())

    def list_config_reports(self, daemon_id: int,
                            limit=10, start=0) -> Optional[ConfigReports]:
        """Lists the config reports for a given daemon. Returns None if the
        review is in progress."""
        params = dict(limit=limit, start=start, id=daemon_id)
        api_instance = ServicesApi(self._api_client)

        # OpenAPI generator doesn't support multiple status codes and empty
        # responses. It expects that the data will always be returned. It is
        # a workaround that adds a string to a list of accepted types. The
        # empty string is received if the status is not equal to 200.
        settings = api_instance.get_daemon_config_reports.settings
        settings['response_type'] = tuple(
            list(settings['response_type']) + [str, ])

        reports, status, _ = api_instance.get_daemon_config_reports(
            **params, _return_http_data_only=False)

        if status == 202:
            return None
        elif status == 204:
            return ConfigReports(total=0, items=[])
        return reports

    def overview(self) -> DhcpOverview:
        """Fetches the DHCP overview. Warning! The OpenAPI client only
        partially deserializes the response. The nested keys don't follow the
        convention, and raw types aren't converted. See Gitlab #727."""
        api_instance = DHCPApi(self._api_client)
        return api_instance.get_dhcp_overview(**self._no_validate_kwargs())

    # Create

    def create_user(self, login: str, email: str, name: str, lastname: str,
                    groups: List[int], password: str) -> User:
        """Creates the user account."""
        user = User(id=0, login=login, email=email, name=name,
                    lastname=lastname, groups=groups)
        account = UserAccount(user, password)
        api_instance = UsersApi(self._api_client)
        return api_instance.create_user(account=account)

    # Read

    def read_machine_state(self, machine_id: int) -> Machine:
        """
        Read the machine state (machine with the additional properties).
        If the machine state wasn't fetched yet then it returns incomplete
        data.
        """
        api_instance = ServicesApi(self._api_client)
        return api_instance.get_machine_state(id=machine_id)

    def read_version(self) -> Version:
        "Read the server version."
        api_instance = GeneralApi(self._api_client)
        return api_instance.get_version()

    # Update

    def update_machine(self, machine: Machine) -> Machine:
        """Updates the machine. It must to contain the valid ID."""
        api_instance = ServicesApi(self._api_client)

        return api_instance.update_machine(
            id=machine["id"],
            machine=machine,
            # This endpoint doesn't return the applications.
            **self._no_validate_kwargs()
        )

    # Complex

    def authorize_all_machines(self) -> Machines:
        """Authorizes all unauthorized machines and returns them."""
        machines = self.list_machines(False)
        machine: Machine
        for machine in machines["items"]:
            machine["authorized"] = True
            self.update_machine(machine)
        return machines

    # Waits

    def _wait_for_event(self,
                        expected_condition: Callable[[Event], bool],
                        **kwargs):
        """Waits for an event that meets the condition."""
        # The last fetch timestamp. It's initialized a minimal date in the UTC
        # timezone.
        fetch_timestamp = datetime(1, 1, 1, tzinfo=timezone.utc)

        @wait_for_success(NoSuccessException,
                          wait_msg="Waiting for an event...")
        def worker():
            nonlocal fetch_timestamp

            # It should list all events, not only the latest. If the events
            # are produced quickly, the expected one may be omitted.
            events = self.list_events(limit=100, **kwargs)
            for event in reversed(events["items"]):
                # Skip older events
                timestamp = event["created_at"]
                if Server._is_before(timestamp, fetch_timestamp):
                    continue
                fetch_timestamp = timestamp

                # Checks if the expected event occurs.
                if expected_condition is not None and expected_condition(event):
                    return
            raise NoSuccessException("expected event doesn't occur")
        return worker()

    def wait_for_next_machine_state(self, machine_id: int,
                                    start: datetime = None, wait_for_apps=True) -> Machine:
        """
        Waits for a next fetch of the machine state after a given date.
        If the date is None then the current moment is used.
        By default,  this function waits until some application is fetched.
        It may be suppressed by specifying a flag.
        """
        if start is None:
            start = datetime.now(timezone.utc)

        @wait_for_success(wait_msg="Waiting to fetch next state...")
        def worker():
            state = self.read_machine_state(machine_id)
            last_visited = state["last_visited_at"]
            if Server._is_before(last_visited, start):
                raise NoSuccessException("the state not fetched")
            if wait_for_apps and len(state["apps"]) == 0:
                raise NoSuccessException("the apps are missing")
            return state
        return worker()

    def wait_for_next_machine_states(self, wait_for_apps=True) -> List[Machine]:
        """
        Waits for the subsequent fetches of the machine states for all machines.
        The machines must be authorized. Returns list of states.
        By default,  this function waits until some application is fetched.
        It may be suppressed by specifying a flag.
        """
        start = datetime.now(timezone.utc)
        machines = self.list_machines(authorized=True)
        states = []
        for machine in machines["items"]:
            state = self.wait_for_next_machine_state(
                machine["id"], start=start, wait_for_apps=wait_for_apps)
            states.append(state)
        return states

    # The different event message is used if the number of subnets is less or
    # greater than 10. Additionally, if the number of subnets is less than 10,
    # each subnet generates its event.
    _pattern_added_subnets = re.compile(
        r'added (?:(?:\d+ subnets)|(?:<subnet.*>)) to <daemon '
        r'id="(?P<daemon_id>\d+)" '
        r'name="(?P<daemon_name>.*)" '
        r'appId="(?P<app_id>\d+)"')

    def wait_for_adding_subnets(self, daemon_id: int = None,
                                daemon_name: str = None, app_id: int = None):
        """Waits for a first adding subnet event that meets the requirements."""
        def condition(ev: Event):
            match = Server._pattern_added_subnets.search(ev["text"])
            if match is None:
                return False
            if daemon_id is not None and \
                    match.group("daemon_id") != str(daemon_id):
                return False
            if daemon_name is not None and \
                    match.group("daemon_name") != daemon_name:
                return False
            if app_id is not None and match.group("app_id") != str(app_id):
                return False
            return True

        self._wait_for_event(condition)

    def wait_for_failed_CA_communication(self, check_unauthorized=True):
        """
        Waits for a failed communication with CA daemon event due to an
        unauthorized server (if needed)."""
        def condition(ev: Event):
            text = ev["text"]
            if not text.startswith("Communication with CA daemon of"):
                return False
            if not text.endswith("failed"):
                return False

            if check_unauthorized and "Unauthorized" not in ev["details"]:
                return False
            return True
        self._wait_for_event(condition)

    def wait_for_update_overview(self) -> DhcpOverview:
        """
        Waits for updating the overview. The overview is up-to-date if all
        local subnet stats are collected after calling this function.
        Warning! It doesn't recognize if the subnet is unavailable (e.g.,
        managed by not running daemon).
        """
        start = datetime.now(timezone.utc)

        @wait_for_success(wait_msg="Waiting to update overview...")
        def worker():
            overview = self.overview()

            subnets: List[Subnet] = []
            if overview["subnets4"]["items"] is not None:
                subnets += overview["subnets4"]["items"]
            if overview["subnets6"]["items"] is not None:
                subnets += overview["subnets4"]["items"]

            for subnet in subnets:
                local_subnet: LocalSubnet
                for local_subnet in subnet["localSubnets"]:
                    collected_at = local_subnet["statsCollectedAt"]
                    if not Server._is_before(collected_at, start):
                        return overview
            raise NoSuccessException("some data are out-of-date")
        return worker()

    @contextmanager
    def no_validate(self):
        """
        Prepares a context where the validation and parsing of the API values
        are disabled. It allows suppressing the errors related to
        non-compliance with the contract Swagger contract.

        Returns
        -------
        This wrapper with disabled input parsing and output validation.

        Examples
        --------
        > server = Server()
        > with server.no_validate() as legacy:
        >     legacy.list_machines()

        Notes
        -----
        It disables the input validation. It causes the parameter names to use
        camelCase instead of snake_case, and timestamps are string instead of
        the Python datetime objects.
        """
        # Suppresses the output validation
        original_validation_rules = self._api_client.configuration.disabled_client_side_validations
        self._api_client.configuration.disabled_client_side_validations = ",".join([
            "multipleOf", "maximum", "exclusiveMaximum", "minimum",
            "exclusiveMinimum", "maxLength", "minLength", "pattern",
            "maxItems", "minItems"
        ])

        original_discard_unknown_types = self._api_client.configuration.discard_unknown_keys
        self._api_client.configuration.discard_unknown_keys = True

        # Suppresses the input parsing and validation - a little hack
        original_call = self._api_client.call_api
        params = dict(_check_type=False)

        def injector(*args, **kwargs):
            kwargs.update(params)
            return original_call(*args, **kwargs)

        self._api_client.call_api = injector

        # Returns the patched wrapper
        yield self

        # Restores the standard behaviour
        self._api_client.call_api = original_call
        self._api_client.configuration.discard_unknown_keys = original_discard_unknown_types
        self._api_client.configuration.disabled_client_side_validations = original_validation_rules
