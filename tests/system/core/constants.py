import os


# The project-specific paths to the docker-compose artifacts.

# Directory with this file (the core module).
_script_dir = os.path.dirname(__file__)

# Directory with the docker-compose file.
_docker_compose_dir = os.path.dirname(_script_dir)

# The configuration directory (absolute path)
_config_directory = os.path.join(_docker_compose_dir, "config")

# === Exported ===

# Docker-compose file.
docker_compose_file = os.path.join(_docker_compose_dir, "docker-compose.yaml")

# The project root directory.
project_directory = os.path.dirname(os.path.dirname(_docker_compose_dir))

# The configuration directory (relative to the project directory path)
config_directory_relative = os.path.relpath(_config_directory, project_directory)
