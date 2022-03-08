# Demo
# Run the demo containers in Docker
# Warning!
# Commands in this file aren't refactored yet! ###

CLEAN.append *FileList["buid-root/**/*"], "build-root"

# premium support
if ENV['cs_repo_access_token']
    ENV['premium'] = 'true'
end
if ENV['premium'] == 'true'
    DOCKER_COMPOSE_FILES='-f docker-compose.yaml -f docker-compose-premium.yaml'
DOCKER_COMPOSE_PREMIUM_OPTS = "--build-arg CS_REPO_ACCESS_TOKEN=#{ENV['cs_repo_access_token']}"
else
    DOCKER_COMPOSE_FILES='-f docker-compose.yaml'
    DOCKER_COMPOSE_PREMIUM_OPTS = ''
end

desc 'Build containers with everything and start all services using docker-compose
arguments: cache=false - forces rebuilding whole container'
task :docker_up => :build_all_in_container do
  at_exit {
    sh "docker-compose #{DOCKER_COMPOSE_FILES} down"
  }
  cache_opt = ''
  if ENV['cache'] == 'false'
    cache_opt = '--no-cache'
  end
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS} #{cache_opt}"
  sh "docker-compose #{DOCKER_COMPOSE_FILES} up"
end

desc 'Build all in container'
task :build_all_in_container do
  sh 'docker/gen-kea-config.py 7000 > docker/kea-dhcp4-many-subnets.conf'
  # we increase the locked memory limit up to 512kb to work around the problem ocurring on Ubuntu 20.04.
  # for details, see: https://github.com/golang/go/issues/35777 and https://github.com/golang/go/issues/37436
  # The workaround added --ulimit memlock=512 to docker build and --privileged to docker run.
  sh "docker build --ulimit memlock=512 -f docker/docker-builder.txt -t stork-builder ."
  sh "docker", "run",
      "-v", "#{ENV["PWD"]}:/repo",
      "--rm", "stork-builder",
      "rake", "build_all_copy_in_subdir"
end

# internal task used by build_all_in_container
task :build_all_copy_in_subdir do
  sh 'mkdir -p ./build-root'
  sh "rsync",
        "-av",
        "--exclude=webui/node_modules",
        "--exclude=webui/dist",
        "--exclude=webui/src/assets/arm",
        "--exclude=webui/src/assets/pkgs",
        "--exclude=doc/_build",
        "--exclude=doc/doctrees",
        "--exclude=backend/server/gen",
        "--exclude=*~",
        "--delete", "api", "backend", "doc", "etc", "grafana", "webui",
        "Rakefile", "rakelib", "./build-root"
  sh "rake install_server[build-root/root] install_agent[build-root/root]"
end

desc 'Shut down all containers'
task :docker_down do
  sh "docker-compose #{DOCKER_COMPOSE_FILES} down"
end

desc 'Build container with Stork Agent and Kea DHCPv4 server'
task :build_kea_container do
  sh 'docker-compose build agent-kea agent-kea-mysql'
end

desc 'Run container with Stork Agent and Kea and mount current Agent binary'
task :run_kea_container do
  at_exit {
    sh 'docker-compose down'
  }
  sh 'docker-compose up --no-deps agent-kea agent-kea-mysql'
end

desc 'Build container with Stork Agent and Kea DHCPv6 server'
task :build_kea6_container do
  sh 'docker-compose build agent-kea6'
end

desc 'Run container with Stork Agent and Kea DHCPv6 server and mount current Agent binary'
task :run_kea6_container do
  at_exit {
    sh 'docker-compose down'
  }
  sh 'docker-compose up agent-kea6'
end

desc 'Build two containers with Stork Agent and Kea HA pair
arguments: cache=false - forces rebuilding whole container'
task :build_kea_ha_containers do
  cache_opt = ''
  if ENV['cache'] == 'false'
    cache_opt = '--no-cache'
  end
  sh "docker-compose build #{cache_opt} agent-kea-ha1 agent-kea-ha2"
end

desc 'Run two containers with Stork Agent and Kea HA pair'
task :run_kea_ha_containers do
  at_exit {
    sh "docker-compose down"
  }
  sh 'docker-compose up agent-kea-ha1 agent-kea-ha2'
end

desc 'Build container with Stork Agent and Kea with host reseverations in db'
task :build_kea_premium_container do
  if not ENV['cs_repo_access_token']
    raise 'ERROR: expected cs_repo_access_token to be set'
  end
  if not File.exist?('build-root')
    raise 'ERROR: build-root not found. Run "rake build_all_in_container" first.'
  end
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS} agent-kea-premium"
end

desc 'Run container with Stork Agent and Kea with host reseverations in db'
task :run_kea_premium_container do
  at_exit {
    sh "docker-compose -f ./docker-compose.yaml -f ./docker-compose-premium.yaml down"
  }
  sh "docker-compose -f ./docker-compose.yaml -f ./docker-compose-premium.yaml up agent-kea-premium hosts-db"
end

desc 'Build container with Stork Agent and BIND 9'
task :build_bind9_container do
  sh 'docker build -f docker/docker-agent-bind9.txt -t agent-bind9 .'
end

desc 'Run container with Stork Agent and BIND 9 and mount current Agent binary'
task :run_bind9_container do
  # host[9999]->agent[8080]
  sh 'docker run --rm -ti -p 9999:8080 --name agent-bind9 -h agent-bind9 -v `pwd`/backend/cmd/stork-agent:/agent agent-bind9'
end

desc 'Build and push demo images'
task :build_and_push_demo_images => :build_all_in_container do
  # build container images with built artifacts
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS}"
  # push built images to docker registry
  sh "docker-compose #{DOCKER_COMPOSE_FILES} push"
end

desc 'Prepare containers that are using in GitLab CI processes'
task :build_ci_containers do
  sh 'docker build --no-cache -f docker/docker-ci-base.txt -t registry.gitlab.isc.org/isc-projects/stork/ci-base:latest docker/'
  #sh 'docker push registry.gitlab.isc.org/isc-projects/stork/ci-base:latest'
end