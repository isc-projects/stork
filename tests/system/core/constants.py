import os


script_dir = os.path.dirname(__file__)
docker_compose_dir = os.path.dirname(script_dir)
project_directory = os.path.dirname(os.path.dirname(docker_compose_dir))
config_directory = os.path.join(docker_compose_dir, "config")
config_directory_relative = os.path.relpath(
    config_directory, project_directory)
