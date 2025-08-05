from datetime import timedelta, datetime
import functools
import logging
import time
import traceback
from typing import Any, Callable, Dict, Hashable
import urllib3


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
TRANSIENT_EXCEPTIONS = (
    TimeoutError,
    ConnectionError,
    urllib3.exceptions.MaxRetryError,
    NoSuccessException,
)
logger = setup_logger(__file__)


def wait_for_success(
    *transient_exceptions,
    wait_msg="Waiting to be ready...",
    sleep_time=timedelta(milliseconds=100),
    max_time=timedelta(seconds=120),
):
    """Wait until function throws no error."""

    transient_exceptions = TRANSIENT_EXCEPTIONS + tuple(transient_exceptions)

    def outer_wrapper(f):
        @functools.wraps(f)
        def inner_wrapper(*args, **kwargs):
            start_time = datetime.now()
            exception = None
            logger.info(wait_msg)
            while True:
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
                elapsed_time = datetime.now() - start_time
                if elapsed_time > max_time:
                    raise TimeoutError(
                        f"Wait time ({max_time.total_seconds()}s) exceeded for {f.__name__}"
                        f" (args: {args}, kwargs {kwargs}). Exception: {exception}"
                    )

        return inner_wrapper

    return outer_wrapper


def tic(label: str = ""):
    """
    A function to quickly measure time between two points in the code.
    The measurements are not reliable for very short time intervals (less than
    500ms).
    Call `tic()` to start the timer. It returns a "toc" function that should be
    called to get the time difference and print elapsed time.
    Accepts an optional label that will be printed with the elapsed time.

    Similarly to the `print` function, the `tic` calls should be removed from
    the production code.

    Example
    -------
    >>> toc = tic("My Timer")
    >>> # ... some code to measure ...
    >>> toc()  # Prints elapsed time for "My Timer"
    """
    if label:
        print("Starting timer for", label)

    start = time.perf_counter()

    def toc():
        end = time.perf_counter()
        elapsed = end - start
        if label:
            print(f"Elapsed time for {label}: {elapsed:.2f} seconds")
        else:
            print(f"Elapsed time: {elapsed:.2f} seconds")

    return toc
