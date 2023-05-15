from core.service_state import ServiceState


def test_is_running():
    # Arrange
    state_running = ServiceState("running", 0, None, None)
    state_starting = ServiceState("starting", 0, None, None)

    # Act & Assert
    assert state_running.is_running()
    assert not state_starting.is_running()


def test_has_healthcheck():
    # Arrange
    state_healthy = ServiceState("running", 0, "healthy", None)
    state_unhealthy = ServiceState("running", 0, "unhealthy", None)
    state_missing = ServiceState("running", 0, None, None)

    # Act & Assert
    assert state_healthy.has_healthcheck()
    assert state_unhealthy.has_healthcheck()
    assert not state_missing.has_healthcheck()


def test_is_heathy():
    # Arrange
    state_healthy = ServiceState("running", 0, "healthy", None)
    state_unhealthy = ServiceState("running", 0, "unhealthy", None)
    state_missing = ServiceState("running", 0, None, None)

    # Act & Assert
    assert state_healthy.is_healthy()
    assert not state_unhealthy.is_healthy()
    assert state_missing.is_healthy()


def test_is_unhealthy():
    # Arrange
    state_healthy = ServiceState("running", 0, "healthy", None)
    state_unhealthy = ServiceState("running", 0, "unhealthy", None)
    state_missing = ServiceState("running", 0, None, None)

    # Act & Assert
    assert not state_healthy.is_unhealthy()
    assert state_unhealthy.is_unhealthy()
    assert not state_missing.is_unhealthy()


def test_is_exited():
    # Arrange
    state_exited = ServiceState("exited", 42, None, None)
    state_running = ServiceState("running", 0, None, None)

    # Act & Assert
    assert state_exited.is_exited()
    assert not state_running.is_exited()


def test_is_starting():
    # Arrange
    state_exited = ServiceState("exited", 42, None, None)
    state_starting = ServiceState("starting", 0, None, None)

    # Act & Assert
    assert not state_exited.is_starting()
    assert state_starting.is_starting()


def test_is_operational():
    # Arrange
    state_starting = ServiceState("starting", 0, None, None)
    state_running = ServiceState("running", 0, None, None)
    state_running_healthy = ServiceState("running", 0, "healthy", None)
    state_running_unhealthy = ServiceState("running", 0, "unhealthy", None)

    # Act & Assert
    assert not state_starting.is_operational()
    assert state_running.is_operational()
    assert state_running_healthy.is_operational()
    assert not state_running_unhealthy.is_operational()


def test_to_string():
    # Arrange
    state_starting = ServiceState("starting", 42, "healthy", "foobar")

    # Act
    string = str(state_starting)

    # Assert
    assert "starting" in string
    assert "healthy" in string
    assert "foobar" not in string
    assert "42" not in string


def test_to_string_unhealthy():
    # Arrange
    state_starting = ServiceState("starting", 42, "unhealthy", "foobar")

    # Act
    string = str(state_starting)

    # Assert
    assert "starting" in string
    assert "healthy" in string
    assert "foobar" in string
    assert "42" not in string


def test_to_string_exited():
    # Arrange
    state_exited = ServiceState("exited", 42, "healthy", "foobar")

    # Act
    string = str(state_exited)

    # Assert
    assert "exited" in string
    assert "healthy" not in string
    assert "foobar" not in string
    assert "42" in string
