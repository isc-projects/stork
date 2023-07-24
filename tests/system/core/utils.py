from datetime import timedelta
import functools
import logging
import time
import traceback
from typing import Any, Callable, Dict, Hashable


def setup_logger(name):
    """Configures the logger with a given name and returns it."""
    logger_instance = logging.getLogger(name)
    logger_instance.setLevel(logging.INFO)
    handler = logging.StreamHandler()
    handler.setLevel(logging.INFO)
    logger_instance.addHandler(handler)
    return logger_instance


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
    memo: Dict[Hashable, Any] = {}

    def wrapper(*args):
        if args in memo:
            return memo[args]
        output = func(*args)
        memo[args] = output
        return output

    return wrapper


class NoSuccessException(Exception):
    """General-purpose exception used by the "wait_for_success" decorator."""


# Get a tuple of transient exceptions for which we'll retry. Other exceptions will be raised.
TRANSIENT_EXCEPTIONS = (TimeoutError, ConnectionError, NoSuccessException)
logger = setup_logger(__file__)


def wait_for_success(
    *transient_exceptions,
    wait_msg="Waiting to be ready...",
    max_tries=120,
    sleep_time: timedelta = timedelta(seconds=1),
):
    """Wait until function throws no error."""

    transient_exceptions = TRANSIENT_EXCEPTIONS + tuple(transient_exceptions)

    def outer_wrapper(f):
        @functools.wraps(f)
        def inner_wrapper(*args, **kwargs):
            exception = None
            logger.info(wait_msg)
            for _ in range(max_tries):
                try:
                    result = f(*args, **kwargs)
                    done_msg = wait_msg + "done"
                    logger.info(done_msg)
                    return result
                except transient_exceptions as ex:
                    logger.debug(
                        "container is not yet ready: %s", traceback.format_exc()
                    )
                    time.sleep(sleep_time.total_seconds())
                    exception = ex
            raise TimeoutError(
                f"Wait time ({max_tries * sleep_time}s) exceeded for {f.__name__}"
                f"(args: {args}, kwargs {kwargs}). Exception: {exception}"
            )

        return inner_wrapper

    return outer_wrapper
