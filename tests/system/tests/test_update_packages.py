from core.wrappers import ExternalPackages
from core.fixtures import external_parametrize


@external_parametrize(version="1.2")
def test_run_old_version_from_packages(external_service: ExternalPackages):
    external_service.log_in_as_admin()
    version = external_service.read_version()["version"]
    assert version == "1.2.0"
