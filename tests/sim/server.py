"""
This module provides a simple interface to the Stork server for use in the
simulator.
TODO: Replace it with OpenAPI client generated from the Stork server API.
"""

import os
import requests


# The Stork server URL. The default value is suitable for the demo environment.
# The environment variable should be set to localhost if the server is running
# on the same host as the simulator.
STORK_SERVER_URL = os.environ.get("STORK_SERVER_URL", "http://server:8080")


def _login_session():
    """Log-in to Stork server as admin with default credentials. Return a
    session object."""
    session = requests.Session()
    credentials = {
        "authenticationMethodId": "ldap",
        "identifier": "admin",
        "secret": "admin",
    }
    session.post(f"{STORK_SERVER_URL}/api/sessions", json=credentials)
    return session


def get_subnets():
    """Fetches the list of subnets from Stork server."""
    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/subnets?start=0&limit=100"
    response = session.get(url)
    data = response.json()

    if data is None or data.get("items") is None:
        return {"items": [], "total": 0}
    return data


def get_bind9_applications():
    """Fetches the list of BIND 9 applications from Stork server."""
    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/apps?app=bind9"
    response = session.get(url)
    data = response.json()

    if data is None or data.get("items") is None:
        return {"items": [], "total": 0}
    return data


def get_machines():
    """Fetches the list of machines from Stork server."""
    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/machines?start=0&limit=100"
    response = session.get(url)
    machines = response.json()
    if machines is None or machines.get("items") is None:
        return {"items": [], "total": 0}
    return machines
