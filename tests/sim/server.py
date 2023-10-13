import os
import requests


STORK_SERVER_URL = os.environ.get("STORK_SERVER_URL", "http://server:8080")


def _login_session():
    session = requests.Session()
    credentials = {
        "authenticationMethodId": "internal",
        "identifier": "admin",
        "secret": "admin",
    }
    session.post(f"{STORK_SERVER_URL}/api/sessions", json=credentials)
    return session


def get_subnets():
    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/subnets?start=0&limit=100"
    response = session.get(url)
    data = response.json()

    if not data:
        return {"items": [], "total": 0}

    for subnet in data["items"]:
        subnet["rate"] = 1
        subnet["clients"] = 1000
        subnet["state"] = "stop"
        subnet["proc"] = None
        if "sharedNetwork" not in subnet:
            subnet["sharedNetwork"] = ""
    return data


def get_applications():
    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/apps/"
    response = session.get(url)
    data = response.json()

    if not data:
        return {"items": [], "total": 0}

    for srv in data["items"]:
        if srv["type"] == "bind9":
            srv["clients"] = 1
            srv["rate"] = 1
            srv["qname"] = "example.com"
            srv["qtype"] = "A"
            srv["transport"] = "udp"
            srv["proc"] = None
            srv["state"] = "stop"
    return data


def get_machines():
    # app.services = {"items": [], "total": 0}

    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/machines?start=0&limit=100"
    response = session.get(url)
    machines = response.json()
    if machines["items"] is None:
        machines["items"] = []
    return machines
