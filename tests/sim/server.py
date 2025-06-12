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

# Once the POST /api/sessions is successful, store the session as a global variable.
stored_session = None


def _login_session():
    """Log-in to Stork server as admin with default credentials. Return a
    session object."""
    from sim import log
    global stored_session
    if stored_session is not None:
        log.info("returning stored session %s", stored_session)
        return stored_session

    requests_session = requests.Session()
    credentials = {
        "authenticationMethodId": "ldap",
        "identifier": "admin",
        "secret": "admin",
    }
    post_session_resp = requests_session.post(f"{STORK_SERVER_URL}/api/sessions", json=credentials, timeout=10)
    if post_session_resp.status_code == 200:
        log.info("successfully logged in")
        stored_session = requests_session
        return requests_session

    raise requests.exceptions.RequestException(f"error creating a session: REST API returned {post_session_resp}")


def get_subnets():
    """Fetches the list of subnets from Stork server."""
    from sim import log
    try:
        session = _login_session()

        url = f"{STORK_SERVER_URL}/api/subnets?start=0&limit=100"
        response = session.get(url)
        data = response.json()

        if data is None or data.get("items") is None:
            return {"items": [], "total": 0}
        return data
    except requests.exceptions.RequestException as err:
        log.error("Error getting subnets: %s", err)
        return {"items": [], "total": 0}
    except BaseException as err:
        log.error("Generic error getting subnets: %s", err)
        return {"items": [], "total": 0}


def get_bind9_applications():
    """Fetches the list of BIND 9 applications from Stork server."""
    from sim import log
    try:
        session = _login_session()

        url = f"{STORK_SERVER_URL}/api/apps?app=bind9"
        response = session.get(url)
        data = response.json()

        if data is None or data.get("items") is None:
            return {"items": [], "total": 0}
        return data
    except requests.exceptions.RequestException as err:
        log.error("Error getting BIND9 apps: %s", err)
        return {"items": [], "total": 0}
    except BaseException as err:
        log.error("Generic error getting BIND9 apps: %s", err)
        return {"items": [], "total": 0}


def get_machines():
    """Fetches the list of machines from Stork server."""
    from sim import log
    try:
        session = _login_session()

        url = f"{STORK_SERVER_URL}/api/machines?start=0&limit=100"
        response = session.get(url)
        machines = response.json()
        if machines is None or machines.get("items") is None:
            return {"items": [], "total": 0}
        return machines
    except requests.exceptions.RequestException as err:
        log.error("Error getting machines: %s", err)
        return {"items": [], "total": 0}
    except BaseException as err:
        log.error("Generic error getting machines: %s", err)
        return {"items": [], "total": 0}
