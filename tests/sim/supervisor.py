from xmlrpc.client import ServerProxy


def get_services(machines):
    data = {"items": [], "total": 0}
    for machine in machines:
        address = machine["address"]
        server = ServerProxy(f"http://{address}:9001/RPC2")
        try:
            services = (
                server.supervisor.getAllProcessInfo()
            )  # pylint: disable=no-member
        except Exception:
            continue
        for srv in services:
            srv["machine"] = address
            data["items"].append(srv)

    data["total"] = len(data["items"])

    # app.services = data

    return data


def stop_service(service):
    server = ServerProxy(f'http://{service["machine"]}:9001/RPC2')
    server.supervisor.stopProcess(service["name"])  # pylint: disable=no-member


def start_service(service):
    server = ServerProxy(f'http://{service["machine"]}:9001/RPC2')
    server.supervisor.startProcess(service["name"])  # pylint: disable=no-member