import logging
from typing import Any, Callable, Hashable


def setup_logger(name):
    logger = logging.getLogger(name)
    logger.setLevel(logging.INFO)
    handler = logging.StreamHandler()
    handler.setLevel(logging.INFO)
    logger.addHandler(handler)
    return logger


def memoize(func: Callable):
    """
    Memoization decorator. Support both functions and methods.

    Parameters
    ----------
    func : Callable
        Function or method that accepts the hashable arguments

    Returns
    -------
    Decorated function/method

    Notes
    -----
    Source: https://stackoverflow.com/a/815160
    """
    memo: dict[Hashable, Any] = {}

    def wrapper(*args):
        if args in memo:
            return memo[args]
        else:
            rv = func(*args)
            memo[args] = rv
            return rv
    return wrapper
