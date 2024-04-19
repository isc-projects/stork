from core.wrappers.agent import Agent


class Bind9(Agent):
    """
    A wrapper for the docker-compose service containing Bind9 and Stork Agent.
    """

    prometheus_exporter_port = 9119
