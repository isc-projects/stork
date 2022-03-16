# Demo
# Run the demo containers in Docker

namespace :docker do

  #################
  ### Functions ###
  #################

  def get_docker_opts(server, cache, services)
    opts = [
      "--project-directory", ".",
      "-f", "docker/docker-compose.yaml"
    ]

    if ENV['CS_REPO_ACCESS_TOKEN']
      opts += ["-f", "docker/docker-compose-premium.yaml"]
    end

    cache_opts = []
    if cache
      cache_opts.append "--no-cache"
    end

    up_opts = [
      "--attach-dependencies",
      "--remove-orphans",
      # Warning! Don't use here "--renew-anon-volumes" options. It causes conflicts between running containers. 
    ]
    additional_services = []

    if server == "local"
      if !services.empty?
        additional_services.append "dns-proxy-server"
      end
      host_server_address = "http://172.20.0.1:8080"
      ENV["STORK_SERVER_URL"] = host_server_address
      up_opts += ["--scale", "server=0", "--scale", "webui=0"]
    elsif server == "ui"
      if !services.empty?
        additional_services.append "webui"
      end
    elsif server == "no-ui"
      if !services.empty?
        additional_services.append "server"
      end
      up_opts += ["--scale", "webui=0"]
    elsif server == "none"
      up_opts += ["--scale", "server=0", "--scale", "webui=0"]
    elsif server == "default" || server == nil
      # Nothing
    else
      puts "Invalid server option. Valid values: 'local', 'ui', 'no-ui', 'none', or empty (keep default). Got: ", server
      fail
    end

    if (services + additional_services).include? "dns-proxy-server"
      opts += ["-f", "docker/docker-compose-dev.yaml"]
    end

    return opts, cache_opts, up_opts, additional_services
  end

  def docker_up_services(server, cache, *services)
    opts, build_opts, up_opts, additional_services = get_docker_opts(server, cache, services)
    # We don't use the BuildKit features in our Dockerfiles (yet).
    # But we turn on the BuildKit to build the Docker stages concurrently and skip unnecessary stages.  
    ENV["COMPOSE_DOCKER_CLI_BUILD"] = "1"
    ENV["DOCKER_BUILDKIT"] = "1"

    sh "docker-compose", *opts, *build_opts, "build", *services, *additional_services
    sh "docker-compose", *opts, "up", *up_opts, *services, *additional_services
  end

  ##################
  ### Demo tasks ###
  ##################

  desc 'Build containers with everything and start all services using docker-compose. Set CS_REPO_ACCESS_TOKEN to use premium features.'
  task :run_all, [:server, :cache] do |t, args|
    docker_up_services(args.server, args.cache == "true")
  end

  desc 'Build and run container with Stork Agent and Kea'
  task :run_kea, [:server, :cache] do |t, args|
    docker_up_services(args.server, args.cache == "true", "agent-kea")
  end

  desc 'Build and run container with Stork Agent and Kea DHCPv6 server'
  task :run_kea6, [:server, :cache] do |t, args|
    docker_up_services(args.server, args.cache == "true", "agent-kea6")
  end

  desc 'Build and run two containers with Stork Agent and Kea HA pair'
  task :run_kea_ha,[:server, :cache] do |t, args|
    docker_up_services(args.server, args.cache == "true", "agent-kea-ha1", "agent-kea-ha2")
  end

  desc 'Build and run container with Stork Agent and Kea with host reseverations in db'
  task :run_kea_premium,[:server, :cache] do |t, args|
    if !ENV["CS_REPO_ACCESS_TOKEN"]
      puts 'You need to provide the CloudSmith access token in CS_REPO_ACCESS_TOKEN environment variable.'
      fail
    end
    docker_up_services(args.server, args.cache == "true", "agent-kea-premium")
  end

  desc 'Build and run container with Stork Agent and BIND 9'
  task :run_bind9,[:server, :cache] do |t, args|
    docker_up_services(args.server, args.cache == "true", "agent-bind9")
  end

  desc 'Build and run container with Postgres'
  task :run_postgres do |t, args|
    docker_up_services("default", false, "postgres")
  end

  desc 'Build and run Docker DNS Proxy Server to resolve internal Docker hostnames'
  # Source: https://stackoverflow.com/a/45071285
  task :run_dns_proxy_server do
    docker_up_services("default", false, "dns-proxy-server")
  end

  desc 'Down all containers and remove all volumes'
  task :down_all do
    ENV["CS_REPO_ACCESS_TOKEN"] = "stub"
    opts, _, _, _ = get_docker_opts(nil, false, [])
    sh "docker-compose", *opts, "down",
        "--volumes",
        "--remove-orphans",
        "--rmi", "local"
  end

  #######################
  ### Docker registry ###
  #######################

  desc 'Prepare containers that are using in GitLab CI processes'
  task :build_ci_containers do
    sh "docker build",
        "--no-cache",
        "-f", "docker/images/ci/ubuntu-18.04.Dockerfile",
        "-t", "registry.gitlab.isc.org/isc-projects/stork/ci-base:latest docker/"
    #sh 'docker push registry.gitlab.isc.org/isc-projects/stork/ci-base:latest'
  end
end

############################################
### Local dev tasks with Docker database ###
############################################

# Internal task to setup access to the Docker database
task :pre_docker_db, [:dbtrace] do |t, args|
  args.with_defaults(
    :dbtrace => "false"
  )

  ENV["STORK_DATABASE_HOST"] = "172.20.0.234"
  ENV["STORK_DATABASE_PORT"] = "5432"
  ENV["STORK_DATABASE_USER_NAME"] = "stork"
  ENV["STORK_DATABASE_PASSWORD"] = "stork"
  ENV["STORK_DATABASE_NAME"] = "stork"
  ENV['PGPASSWORD'] = "stork"
end

desc 'Run local server with Docker database'
task :run_server_db, [:dbtrace] => [:pre_docker_db] do |t, args|
  Rake::MultiTask.new(:stub, t.application)
    .enhance([:run_server, "docker:run_postgres"])
    .invoke()
end

desc 'Run local unittests with Docker database'
task :unittest_backend_db, [:dbtrace] => [:pre_docker_db] do |t, args|
  Rake::MultiTask.new(:stub, t.application)
    .enhance([:unittest_backend, "docker:run_postgres"])
    .invoke()
end