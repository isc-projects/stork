import re
from contextlib import contextmanager
from datetime import datetime, timezone
from typing import Callable, List, Optional, TypeVar

import openapi_client
from core.compose import DockerCompose
from core.utils import NoSuccessException, wait_for_success
from core.wrappers.compose import ComposeServiceWrapper
from openapi_client.api.dhcp_api import DHCPApi, DhcpOverview, Hosts, Leases, Subnets
from openapi_client.api.dns_api import DNSApi
from openapi_client.api.events_api import Events, EventsApi
from openapi_client.api.general_api import GeneralApi, Version
from openapi_client.api.services_api import (
    ConfigReports,
    Machine,
    Machines,
    ServicesApi,
)
from openapi_client.api.settings_api import Settings, SettingsApi
from openapi_client.api.users_api import Groups, User, UserAccount, Users, UsersApi
from openapi_client.models.create_host_begin_response import CreateHostBeginResponse
from openapi_client.models.update_host_begin_response import UpdateHostBeginResponse
from openapi_client.models.event import Event
from openapi_client.models.host import Host
from openapi_client.models.puller import Puller
from openapi_client.exceptions import ServiceException
from openapi_client.models.zone_inventory_states import ZoneInventoryStates
from openapi_client.models.zones import Zones


T1 = TypeVar("T1")
T2 = TypeVar("T2")


class Server(ComposeServiceWrapper):  # pylint: disable=too-many-public-methods)
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
        url = f"http://{mapped[0]}:{mapped[1]}/api"
        configuration = openapi_client.Configuration(host=url)
        self._api_client = openapi_client.ApiClient(configuration)
        self._server_supervisor_service = self._get_supervisor_service("stork-server")

    def close(self):
        """Free the resources used by the wrapper."""
        self._api_client = None

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

    @staticmethod
    def _parse_date(date):
        """
        Parses the GO timestamp if it is a string otherwise, lefts it as is.
        """
        if isinstance(date, str):
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

    # Process

    def get_stork_server_pid(self):
        """Returns PID of the stork-server process."""
        return self._server_supervisor_service.get_pid()

    def reload_stork_server(self):
        """Sends SIGHUP to the stork-server."""
        self._server_supervisor_service.reload()

    def restart_stork_server(self):
        """
        Restarts the Stork server and waits to recover an operational status.
        """
        self._server_supervisor_service.restart()

    def interrupt_stork_server(self):
        """Sends SIGINT to the stork-server."""
        self._server_supervisor_service.interrupt()

    # Authentication

    def log_in(
        self, username: str, password: str, authentication_method_id="internal"
    ) -> User:
        """Logs in a user. Returns the user info."""
        api_instance = UsersApi(self._api_client)
        http_info = api_instance.create_session_with_http_info(
            credentials={
                "identifier": username,
                "secret": password,
                "authentication_method_id": authentication_method_id,
            },
        )
        headers = http_info.headers
        user = http_info.data

        session_cookie = headers["Set-Cookie"]
        self._api_client.cookie = session_cookie
        return user

    def log_in_as_admin(self):
        """Logs in an admin. Returns the user info."""
        return self.log_in("admin", "admin")

    def log_out(self):
        """Logs out the current user."""
        api_instance = UsersApi(self._api_client)
        api_instance.delete_session()
        self._api_client.cookie = None

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
        params = {"start": start, "limit": limit}
        if authorized is not None:
            params["authorized"] = authorized
        api_instance = ServicesApi(self._api_client)
        return api_instance.get_machines(**params)

    def list_subnets(
        self, app_id=None, family: int = None, limit=10, start=0
    ) -> Subnets:
        """Lists the subnets from a given application and/or family."""
        params = {"start": start, "limit": limit}
        if app_id is not None:
            params["app_id"] = app_id
        if family is not None:
            params["dhcp_version"] = family

        api_instance = DHCPApi(self._api_client)
        return api_instance.get_subnets(**params)

    def list_events(
        self,
        daemon_type=None,
        app_type=None,
        machine_id=None,
        user_id=None,
        limit=10,
        start=0,
    ) -> Events:
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
        params = {"start": start, "limit": limit}
        if daemon_type is not None:
            params["daemon_type"] = daemon_type
        if app_type is not None:
            params["app_type"] = app_type
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
        if text is not None:
            params["text"] = text
        if host_id is not None:
            params["host_id"] = host_id

        api_instance = DHCPApi(self._api_client)
        return api_instance.get_leases(**params)

    def list_hosts(self, text=None) -> Hosts:
        """Lists the hosts based on the host identifier."""
        params = {"limit": 100}
        if text is not None:
            params["text"] = text
        api_instance = DHCPApi(self._api_client)
        return api_instance.get_hosts(**params)

    def list_config_reports(
        self, daemon_id: int, limit=10, start=0
    ) -> Optional[ConfigReports]:
        """Lists the config reports for a given daemon. Returns None if the
        review is in progress."""
        params = {"start": start, "limit": limit, "id": daemon_id}
        api_instance = ServicesApi(self._api_client)

        http_info = api_instance.get_daemon_config_reports_with_http_info(
            **params,
        )

        status = http_info.status_code
        reports = http_info.data

        if status == 202:
            return None
        if status == 204:
            return ConfigReports(total=0, items=[])
        return reports

    def overview(self) -> DhcpOverview:
        """Fetches the DHCP overview. Warning! The OpenAPI client only
        partially deserializes the response. The nested keys don't follow the
        convention, and raw types aren't converted. See Gitlab #727."""
        api_instance = DHCPApi(self._api_client)
        return api_instance.get_dhcp_overview()

    # Create

    def create_user(self, user: User, password: str) -> User:
        """Creates the user account."""
        user.id = 0
        account = UserAccount.from_dict(
            {
                "user": user,
                "password": password,
            }
        )
        api_instance = UsersApi(self._api_client)
        return api_instance.create_user(account=account)

    def create_host_reservation(self, host: Host):
        """Shorthand to add a host reservation."""
        with self.transaction_create_host_reservation() as (_, submit, _):
            submit(host)

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
        """Read the server version."""
        api_instance = GeneralApi(self._api_client)
        return api_instance.get_version()

    def read_puller(self, puller_id) -> Puller:
        """Read the puller state"""
        api_instance = SettingsApi(self._api_client)
        return api_instance.get_puller(id=puller_id)

    def read_settings(self) -> Settings:
        """Read server configuration settings."""
        api_instance = SettingsApi(self._api_client)
        return api_instance.get_settings()

    def read_server_token(self) -> str:
        """Read server token."""
        api_instance = ServicesApi(self._api_client)
        response = api_instance.get_machines_server_token()
        return response.token

    def read_access_point_key(self, app_id) -> str:
        """Read the access point key."""
        api_instance = ServicesApi(self._api_client)
        # Currently, only the control access points support the key.
        response = api_instance.get_access_point_key(app_id=app_id, type="control")
        return response

    def get_zone_inventory_states(self) -> ZoneInventoryStates:
        """Get zone inventory states."""
        api_instance = DNSApi(self._api_client)

        http_info = api_instance.get_zone_inventory_states_with_http_info()

        status = http_info.status_code
        zone_inventory_states = http_info.data

        if status == 202:
            return None
        if status == 204:
            return ZoneInventoryStates(total=0, items=[])
        return zone_inventory_states

    def get_zones(self, start: int, limit: int) -> Zones:
        """Get zones with paging."""
        api_instance = DNSApi(self._api_client)
        return api_instance.get_zones(start, limit)

    # Update

    def update_machine(self, machine: Machine) -> Machine:
        """Updates the machine. It must to contain the valid ID."""
        api_instance = ServicesApi(self._api_client)

        # This endpoint doesn't return the applications.
        return api_instance.update_machine(
            id=machine.id,
            machine=machine,
        )

    def update_host_reservation(self, host: Host):
        """Shorthand to update a host reservation."""
        with self.transaction_update_host_reservation(host.id) as (_, submit, _):
            submit(host)

    def update_settings(self, settings: Settings):
        """Updates the server configuration settings in the database."""
        api_instance = SettingsApi(self._api_client)
        api_instance.update_settings(settings)

    # Delete

    @wait_for_success(
        ServiceException, wait_msg="Waiting to delete machine...", max_tries=5
    )
    def delete_machine(self, machine_id: int):
        """
        Deletes a machine and references.

        A deadlock may occur if the config review is in progress. The waiting
        decorator is used as a workaround.
        """
        api_instance = ServicesApi(self._api_client)
        return api_instance.delete_machine(id=machine_id)

    # Background

    def fetch_zones(self):
        """Begins fetching the zones from the monitored servers."""
        api_instance = DNSApi(self._api_client)
        api_instance.put_zones_fetch()

    # Transactional

    def transaction_create_host_reservation(self):
        """Creates a transaction context to add the host reservation."""
        api_instance = DHCPApi(self._api_client)

        def on_begin() -> CreateHostBeginResponse:
            # Begin transaction response contains the daemons without related
            # app access points.
            return api_instance.create_host_begin()

        def on_submit(transaction_id: int, host: Host):
            api_instance.create_host_submit(id=transaction_id, host=host)

        def on_cancel(transaction_id: int):
            api_instance.create_host_delete(id=transaction_id)

        return Server._api_transaction(on_begin, on_submit, on_cancel)

    def transaction_update_host_reservation(self, host_id: int):
        """Creates a transaction context to update the host reservation."""
        api_instance = DHCPApi(self._api_client)

        def on_begin() -> UpdateHostBeginResponse:
            # Begin transaction response contains the daemons without related
            # app access points.
            return api_instance.update_host_begin(host_id=host_id)

        def on_submit(transaction_id: int, host: Host):
            api_instance.update_host_submit(
                id=transaction_id, host_id=host_id, host=host
            )

        def on_cancel(transaction_id: int):
            api_instance.update_host_delete(id=transaction_id, host_id=host_id)

        return Server._api_transaction(on_begin, on_submit, on_cancel)

    @staticmethod
    @contextmanager
    def _api_transaction(
        on_begin: Callable[[], T1],
        on_submit: Callable[[int, T2], None],
        on_cancel: Callable[[int], None],
        transaction_id_extractor: Callable[[T1], int] = lambda ctx: ctx.id,
    ):
        """
        Helper function to send requests in the transaction context. If the
        transaction is not submitted, it will be canceled automatically.

        Parameters
        ----------
        on_begin: callable
            Called to start the transaction. It accepts no arguments and
            returns the transaction context.
        on_submit: callable
            Called to submit the transaction. It accepts the transaction ID and
            submitted data. Returns nothing.
        on_cancel: callable
            Called to cancel the transaction. It accepts the transaction ID.
            Returns nothing.
        transaction_id_extractor: callable
            Extracts the transaction ID from the transaction context. By
            default uses the ID property.
        """

        ctx = on_begin()
        transaction_id = transaction_id_extractor(ctx)

        state = "pending"

        def cancel():
            nonlocal state
            if state == "canceled":
                raise Exception("transaction already canceled")
            if state == "submitted":
                raise Exception("transaction already submitted")

            on_cancel(transaction_id)
            state = "canceled"

        def submit(data: T2):
            nonlocal state
            if state == "canceled":
                raise Exception("transaction already canceled")
            if state == "submitted":
                raise Exception("transaction already submitted")

            on_submit(transaction_id, data)
            state = "submitted"

        yield (ctx, submit, cancel)

        if state == "pending":
            cancel()

    # Complex

    def authorize_all_machines(self) -> Machines:
        """Authorizes all unauthorized machines and returns them."""
        machines = self.list_machines(False)
        for machine in machines.items:
            machine.authorized = True
            self.update_machine(machine)
        return machines

    # Waits

    def _wait_for_event(self, expected_condition: Callable[[Event], bool], **kwargs):
        """Waits for an event that meets the condition."""
        # The last fetch timestamp. It's initialized a minimal date in the UTC
        # timezone.
        fetch_timestamp = datetime(1, 1, 1, tzinfo=timezone.utc)

        @wait_for_success(NoSuccessException, wait_msg="Waiting for an event...")
        def worker():
            nonlocal fetch_timestamp

            # It should list all events, not only the latest. If the events
            # are produced quickly, the expected one may be omitted.
            events = self.list_events(limit=100, **kwargs)
            for event in reversed(events.items):
                # Skip older events
                timestamp = event.created_at
                if Server._is_before(timestamp, fetch_timestamp):
                    continue
                fetch_timestamp = timestamp

                # Checks if the expected event occurs.
                if expected_condition is not None and expected_condition(event):
                    return
            raise NoSuccessException("expected event doesn't occur")

        return worker()

    def _wait_for_puller(self, puller_id: str, start: datetime = None):
        """
        Waits for finishing the next execution of a given puller.

        It uses invoked and finished time to ensure that pulling was completely
        performed after the start time.

        The below example shows the three sequential pulls on the timeline.
        The given start time is marked on the timeline. The 'I' letter means
        an invoked time for a specific pull and the 'E' letter the execution
        time.

        Pull 1 is completely performed before the start time.

        Pull 2 began before the start time. If we consider only the finish
        time, the waiting lock will be released when this pull is ended. If
        any change was provided between the invoked time of pull and start time,
        it would not be reflected yet when the waiting lock is gone.

        Only pull 3 guarantees that the pulled data will be updated after the
        start time because the pull was invoked after the start time, and the
        finish time is after the invoked time.

                              start
        time   .................|.................
        pull 1    I######E
        pull 2              I######E
        pull 3                        I######E

        """
        if start is None:
            start = datetime.now(timezone.utc)

        # Prepares a name used to logging.
        interval_suffix = "_interval"
        friendly_puller_name = puller_id
        if puller_id.endswith(interval_suffix):
            friendly_puller_name = puller_id[: -len(interval_suffix)]

        pulling_started_at = None

        @wait_for_success(
            wait_msg=f'Waiting to next "{friendly_puller_name}" execution...'
        )
        def worker():
            nonlocal pulling_started_at

            puller = self.read_puller(puller_id)

            last_invoked_at = puller.last_invoked_at
            last_finished_at = puller.last_finished_at

            # Wait for start new pulling
            if pulling_started_at is None:
                if Server._is_before(last_invoked_at, start):
                    raise NoSuccessException("the puller not invoked")
                pulling_started_at = last_invoked_at

            # Wait for finish pulling
            if Server._is_before(last_finished_at, pulling_started_at):
                raise NoSuccessException("the puller not finished")

        return worker()

    def wait_for_host_reservation_pulling(self, start: datetime = None):
        """Waits for finishing the next execution of host reservation puller."""
        return self._wait_for_puller("kea_hosts_puller_interval", start)

    def wait_for_kea_statistics_pulling(self, start: datetime = None):
        """Waits for finishing next execution of Kea statistics puller."""
        return self._wait_for_puller("kea_stats_puller_interval", start)

    def wait_for_bind9_statistics_pulling(self, start: datetime = None):
        """Waits for finishing next execution of Bind9 statistics puller."""
        return self._wait_for_puller("bind9_stats_puller_interval", start)

    def _wait_for_states_pulling(self, start: datetime = None):
        """
        Waits for finishing next execution of application state puller. Unlike
        the `last_visited_at` property from the application entry, it waits
        until the end of writing all application-related entries (subnets,
        shared networks, hosts) to the database.
        """
        self._wait_for_puller("apps_state_puller_interval", start)

    @wait_for_success(wait_msg="Waiting to fetch next machine state...")
    def wait_for_next_machine_state(
        self, machine_id: int, start: datetime = None, wait_for_apps=True
    ) -> Machine:
        """
        Waits for a next fetch of the machine state after a given date.
        If the date is None then the current moment is used.
        By default, this function waits until some application is fetched.
        It may be suppressed by specifying a flag.
        """
        self._wait_for_states_pulling(start)
        state = self.read_machine_state(machine_id)
        if wait_for_apps and len(state.apps) == 0:
            raise NoSuccessException("the apps are missing")
        return state

    @wait_for_success(wait_msg="Waiting to fetch next machine states...")
    def wait_for_next_machine_states(
        self, start: datetime = None, wait_for_apps=True
    ) -> List[Machine]:
        """
        Waits for the subsequent fetches of the machine states for all machines.
        The machines must be authorized. Returns list of states.
        By default, this function waits until some application is fetched.
        It may be suppressed by specifying a flag.
        """
        self._wait_for_states_pulling(start)
        machines = self.list_machines(authorized=True)
        states = []
        for machine in machines.items:
            state = self.read_machine_state(machine.id)
            if wait_for_apps and len(state.apps) == 0:
                raise NoSuccessException("the apps are missing")
            states.append(state)
        return states

    # The different event message is used if the number of subnets is less or
    # greater than 10. Additionally, if the number of subnets is less than 10,
    # each subnet generates its event.
    _pattern_added_subnets = re.compile(
        # pylint: disable=implicit-str-concat
        r"added (?:(?:\d+ subnets)|(?:<subnet.*>)) to <daemon "
        r'id="(?P<daemon_id>\d+)" '
        r'name="(?P<daemon_name>.*)" '
        r'appId="(?P<app_id>\d+)"'
    )

    def wait_for_failed_ca_communication(self, check_unauthorized=True):
        """
        Waits for a failed communication with CA daemon event due to an
        unauthorized server (if needed).
        """

        def condition(event: Event):
            text = event.text
            if not text.startswith("forwarding Kea command via"):
                return False
            if not text.endswith("failed"):
                return False

            if check_unauthorized and "Unauthorized" not in event.details:
                return False
            return True

        self._wait_for_event(condition)

    @wait_for_success(wait_msg="Waiting for config reports...")
    def wait_for_config_reports(
        self, daemon_id: int, limit: int = 100, start: int = 0
    ) -> ConfigReports:
        """Waits for finishing the current config review."""
        reports = self.list_config_reports(daemon_id, limit=limit, start=start)
        if reports is None:
            raise NoSuccessException("reviews aren't ready yet")
        return reports

    @wait_for_success(wait_msg="Waiting for HA peers to be ready...")
    def wait_for_ha_ready(self):
        """
        Waits for accomplishing synchronizing the HA pair.
        The HA peers must be detected and authorized.
        Supports only hot-standby mode.
        """
        valid_states = ["hot-standby"]

        overview = self.overview()

        for daemon in overview.dhcp_daemons:
            for relationship in daemon.ha_overview:
                if relationship.ha_state not in valid_states:
                    identifier = f"{daemon.app_name}@{daemon.machine}/{daemon.name}"
                    raise NoSuccessException(
                        f"The {identifier} HA peer is {relationship.ha_state}"
                    )

    @wait_for_success(wait_msg="Waiting for zones fetching completion...")
    def wait_for_fetch_zones(self) -> ZoneInventoryStates:
        """Waits for finishing the zones fetch."""
        zones_states = self.get_zone_inventory_states()
        if zones_states is None:
            raise NoSuccessException("zones have not been fetched yet")
        return zones_states
