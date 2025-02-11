# Demo
# Run the demo containers in Docker

namespace :demo do
    ALL_DEMO_COMPOSE_FILES = FileList["docker/docker-compose*.yaml"]

    # Produces the arguments for docker-compose.
    # Parameters:
    # Server_mode - server service mode, possible values:
    #     - host (doesn't start server container, but uses locally running one on host)
    #     - with-ui (run server and webui)
    #     - without-ui (run server but no webui)
    #     - no-server (no server and no webui)
    #     - default
    # Cache - doesn't rebuild the container
    # Services - list of service names; if empty then all services are used
    # Detach - run services in the detached mode
    # Environment variables:
    # CS_REPO_ACCESS_TOKEN - CloudSmith repo token, required for premium services
    def get_docker_opts(server_mode, cache, detach, services)
        opts = [
            "--project-directory", ".",
            "-f", "docker/docker-compose.yaml"
        ]

        if ENV['CS_REPO_ACCESS_TOKEN']
            opts += ["-f", "docker/docker-compose-premium.yaml"]
        end

        cache_opts = []
        if !cache
            cache_opts.append "--no-cache"
        end

        up_opts = []
        if detach
            up_opts.append "-d"
        else
            # Warning! Don't use here "--renew-anon-volumes" options. It causes conflicts between running containers.
            up_opts.append "--attach-dependencies", "--remove-orphans"
        end

        additional_services = []

        if server_mode == "host"
            if !check_hosts_and_print_hint(ALL_DEMO_COMPOSE_FILES)
                fail "Update the /etc/hosts file"
            end
            host_server_address = "http://host.docker.internal:8080"
            if OS == "linux"
                host_server_address = "http://172.24.0.1:8080"
            end
            ENV["STORK_SERVER_URL"] = host_server_address
            if services.empty? || services.include?("server")
                up_opts.append "--scale", "server=0"
            end
            if services.empty? || services.include?("webui")
                up_opts.append "--scale", "webui=0"
            end
        elsif server_mode == "with-ui"
            if !services.empty?
                additional_services.append "webui"
            end
        elsif server_mode == "without-ui"
            if !services.empty?
                additional_services.append "server"
            end
            if services.empty? || services.include?("webui")
                up_opts.append "--scale", "webui=0"
            end
        elsif server_mode == "no-server"
            if services.empty? || services.include?("server")
                up_opts.append "--scale", "server=0"
            end
            if services.empty? || services.include?("webui")
                up_opts.append "--scale", "webui=0"
            end
            # Prevents the Stork Agent from the registration
            ENV["STORK_SERVER_URL"] = ""
        elsif server_mode == "default" || server_mode == nil
            # Nothing
        else
            puts "Invalid server mode option. Valid values: 'host', 'with-ui', 'without-ui', 'no-server', or empty (keep default). Got: ", server
            fail
        end

        return opts, cache_opts, up_opts, additional_services
    end

    # Calls docker-compose up command for the given services, uses all services
    # if the input list is empty
    # SERVER_MODE - server mode - choice: host, with-ui, without-ui, no-server, default
    # CACHE - doesn't rebuild the containers if present - default: true
    # DETACH - run services in detached mode - default: false
    def docker_up_services(*services)
        # Read arguments from the environment variables
        server_mode = ENV["SERVER_MODE"]
        cache = ENV["CACHE"] != "false"
        detach = ENV["DETACH"] == "true"

        # Prepare the docker-compose flags
        opts, build_opts, up_opts, additional_services = get_docker_opts(server_mode, cache, detach, services)

        # We don't use the BuildKit features in our Dockerfiles (yet).
        # But we turn on the BuildKit to build the Docker stages concurrently and skip unnecessary stages.
        ENV["COMPOSE_DOCKER_CLI_BUILD"] = "1"
        ENV["DOCKER_BUILDKIT"] = "1"

        # Execute the docker-compose commands
        sh *DOCKER_COMPOSE, *opts, "build", *build_opts, *services, *additional_services
        sh *DOCKER_COMPOSE, *opts, "up", *up_opts, *services, *additional_services
    end

    ##################
    ### Demo tasks ###
    ##################

    desc 'Build containers with everything and start all services using docker-compose. Set CS_REPO_ACCESS_TOKEN to use premium features.
    SERVER_MODE - Server mode - choice: host, with-ui, without-ui, no-server, default, default: default
    host - Do not run the server in Docker, instead use the local one (which must be run separately on host)
    with-ui - Run server in Docker with UI
    without-ui - Run server in Docker without UI
    no-server - Suppress running the server service
    default - Use default service configuration from the compose file (default)
    CACHE - Use the Docker cache - default: true
    '
    task :up => [DOCKER_COMPOSE] do
        docker_up_services()
    end

    namespace :up do
        desc 'Build and run container with Stork Agent and Kea
        See "up" command for arguments.'
        task :kea => [DOCKER_COMPOSE] do
            docker_up_services("agent-kea")
        end

        desc 'Build and run container with Stork Agent and Kea with many subnets in the configuration.
        See "up" command for arguments.'
        task :kea_many_subnets => [DOCKER_COMPOSE] do
            docker_up_services("agent-kea-many-subnets")
        end

        desc 'Build and run container with Stork Agent and Kea DHCPv6 server
        See "up" command for arguments.'
        task :kea6 => [DOCKER_COMPOSE] do
            docker_up_services("agent-kea6")
        end

        desc 'Build and run three containers with Stork Agent and Kea HA pair
        See "up" command for arguments.'
        task :kea_ha => [DOCKER_COMPOSE] do
            docker_up_services("agent-kea-ha1", "agent-kea-ha2", "agent-kea-ha3")
        end

        desc 'Build and run container with Stork Agent and Kea with host reservations in db
        CS_REPO_ACCESS_TOKEN - CloudSmith token - required
        See "up" command for more arguments.'
        task :kea_premium => [DOCKER_COMPOSE] do
            if !ENV["CS_REPO_ACCESS_TOKEN"]
                fail 'You need to provide the CloudSmith access token in CS_REPO_ACCESS_TOKEN environment variable.'
            end
            docker_up_services("agent-kea-premium-one", "agent-kea-premium-two")
        end

        desc 'Build and run container with Stork Agent and BIND 9
        See "up" command for arguments.'
        task :bind9 => [DOCKER_COMPOSE] do
            docker_up_services("agent-bind9")
        end

        desc 'Build and run container with Stork Agent and PowerDNS
        See "up" command for arguments.'
        task :pdns => [DOCKER_COMPOSE] do
            docker_up_services("agent-pdns")
        end

        desc 'Build and run container with Postgres
            POSTGRES_VERSION - use specific Postgres database version - optional'
        task :postgres => [DOCKER_COMPOSE] do
            docker_up_services("postgres")
        end

        desc 'Build and run container with OpenLDAP'
        task :ldap => [DOCKER_COMPOSE] do
            docker_up_services("openldap")
        end

        desc 'Build and run Docker DNS Proxy Server to resolve internal Docker hostnames'
        # Source: https://stackoverflow.com/a/45071285
        task :dns_proxy_server => [DOCKER_COMPOSE] do
            docker_up_services("dns-proxy-server")
        end

        desc 'Build and run container with simulator'
        task :simulator => [DOCKER_COMPOSE] do
            docker_up_services("simulator")
        end

        desc 'Build and run web UI served by Nginx and Apache in separate containers'
        task :webui => [DOCKER_COMPOSE] do
            docker_up_services("webui", "webui-apache")
        end

        desc 'Build and run Grafana and Prometheus containers'
        task :grafana => [DOCKER_COMPOSE] do
            docker_up_services("grafana")
        end
    end

    desc 'Down all containers and remove all volumes'
    task :down => [DOCKER_COMPOSE] do
        ENV["CS_REPO_ACCESS_TOKEN"] = "stub"
        opts, _, _, _ = get_docker_opts(nil, false, false, [])
        sh *DOCKER_COMPOSE, *opts, "down",
            "--volumes",
            "--remove-orphans",
            "--rmi", "local"
    end

    desc 'Checks the /etc/hosts file content'
    task :check_etchosts do
        check_hosts_and_print_hint(ALL_DEMO_COMPOSE_FILES)
    end

    desc 'Print logs of a given service
        SERVICE - service name - optional'
    task :logs => [DOCKER_COMPOSE] do
        ENV["CS_REPO_ACCESS_TOKEN"] = "stub"
        opts, _, _, _ = get_docker_opts(nil, false, false, [])
        services = []
        if !ENV["SERVICE"].nil?
            services.append ENV["SERVICE"]
        end
        sh *DOCKER_COMPOSE, *opts, "logs", *services
    end

    desc 'Run shell inside specific service
        SERVICE - service name - required
        SERVICE_USER - user to login - optional, default: root'
    task :shell => [DOCKER_COMPOSE] do
        ENV["CS_REPO_ACCESS_TOKEN"] = "stub"
        opts, _, _, _ = get_docker_opts(nil, false, false, [])
        exec_opts = []
        if !ENV["SERVICE_USER"].nil?
            exec_opts.append "--user", ENV["SERVICE_USER"]
        end
        sh *DOCKER_COMPOSE, *opts, "exec", *exec_opts, ENV["SERVICE"], "/bin/sh"
    end

    desc "Build the demo containers
        CS_REPO_ACCESS_TOKEN - CloudSmith token - optional
        SERVICE - service name - optional
        CACHE - doesn't rebuild the containers if present - default: true"
    task :build => [DOCKER_COMPOSE] do
        services = []
        if !ENV["SERVICE"].nil?
            services.append ENV["SERVICE"]
        end

        cache = ENV["CACHE"] != "false"

        # Prepare the docker-compose flags
        opts, build_opts, _, additional_services = get_docker_opts(nil, cache, false, services)

        # We don't use the BuildKit features in our Dockerfiles (yet).
        # But we turn on the BuildKit to build the Docker stages concurrently
        # and skip unnecessary stages.
        ENV["COMPOSE_DOCKER_CLI_BUILD"] = "1"
        ENV["DOCKER_BUILDKIT"] = "1"

        sh *DOCKER_COMPOSE, *opts, "build", *build_opts, *services, *additional_services
    end

    desc "Collects the performance data from the demo containers and generates
        the report
        TMP_DIRECTORY - temporary directory to store the performance data - optional, default: system temporary directory"
    task :performance => [DOCKER_COMPOSE, PYTHON, PYTEST] do
        # Fetch running services.
        opts, _, _, _ = get_docker_opts(nil, false, false, [])
        services = []
        services_text, _, _ = Open3.capture3 *DOCKER_COMPOSE, *opts, "ps", "--services"

        # Do in a temporary directory.
        #
        # The temporary directory is customizable as a workaround for the
        # Internet browsers installed from Ubuntu snap packages. The snap
        # packages are running in a sandbox and they cannot access the /tmp
        # directory. It blocks the opening of the generated performance report.
        require 'tmpdir'
        Dir.mktmpdir(nil, ENV["TMP_DIRECTORY"]) do |dir|
            # For each service, copy the performance data.
            services_text.split("\n").each do |service|
                # Copy the performance data from the containers.
                data_path = "/var/log/supervisor/performance-report"
                # The performance data are not available in all containers.
                stdout, stderr, status = Open3.capture3 *DOCKER_COMPOSE, *opts,
                    "cp",
                    "#{service}:#{data_path}",
                    "#{dir}/#{service}.data0"

                if status != 0
                    puts "No performance data for #{service}"
                    next
                end

                # Fetch the rotation data if available.
                Open3.capture3 *DOCKER_COMPOSE, *opts,
                    "cp",
                    "#{service}:#{data_path}.old",
                    "#{dir}/#{service}.data1"
            end

            # Generate the report.
            report_path = File.join dir, "performance_report.html"
            sh *PYTHON, "tests/system/core/performance_chart.py",
                "--output", report_path,
                *FileList[File.join(dir, "*.data?")]

            # Open the report.
            open_file report_path

            # Wait for key press to clean up the performance data.
            require 'io/console'
            puts ">>> Press any key to clean up the performance data <<<"
            STDIN.getch
        end
    end
end

############################################
### Local dev tasks with Docker database ###
############################################

# Internal task to setup access to the Docker database
# DB_TRACE - trace SQL queries - default: false
task :pre_docker_db do
    if ENV["DB_TRACE"] == "true"
        ENV["STORK_DATABASE_TRACE"] = "run"
    end

    # Uses exposed port
    ENV["STORK_DATABASE_HOST"] = "localhost"
    ENV["STORK_DATABASE_PORT"] = "5678"
    ENV["STORK_DATABASE_USER_NAME"] = "stork"
    ENV["STORK_DATABASE_PASSWORD"] = "stork"
    ENV["STORK_DATABASE_NAME"] = ENV["STORK_DATABASE_NAME"] || "stork"
    ENV["PGPASSWORD"] = "stork"
    ENV["STORK_DATABASE_MAINTENANCE_NAME"] = "stork"
    ENV["STORK_DATABASE_MAINTENANCE_USER_NAME"] = "stork"
    ENV["STORK_DATABASE_MAINTENANCE_PASSWORD"] = "stork"
    # Environment variable to skip some unit tests not working under Docker.
    ENV["STORK_DATABASE_IN_DOCKER"] = "true"
end

# Waits for a given docker-compose service be operational (Up and Healthy status)
def wait_to_be_operational(service)
    opts, _, _, _ = get_docker_opts(nil, false, false, [service])
    contener_id, _, status = Open3.capture3 *DOCKER_COMPOSE, *opts, "ps", "-q"
    if status != 0
        fail "Unknown container"
    end
    container_id = contener_id.rstrip
    wait_time = 2
    retries = 10
    attempt = 0

    loop do
        status_text, _, _ = Open3.capture3(
            DOCKER, "ps", "--format", "{{ .Status }}",
            "-f", "id=" + container_id
        )
        status_text = status_text.rstrip
        is_operational = status_text.start_with?("Up") && status_text.include?("healthy")
        break if is_operational
        break if attempt == retries

        sleep wait_time
        attempt += 1
        print "Wait for ", service, " to be operational... ", attempt, "/", retries, "\n"
    end

    if attempt == retries
        fail "Maximum number of retries exceed."
    end
end


namespace :run do
    desc 'Run local server with Docker database
    DB_TRACE - trace SQL queries - default: false'
    task :server_db => [:pre_docker_db] do
        at_exit {
            Rake::Task["demo:down"].invoke()
        }

        ENV["DETACH"] = "true"
        Rake::Task["demo:up:postgres"].invoke()
        wait_to_be_operational("postgres")
        Rake::Task["run:server"].invoke()
    end
end

namespace :unittest do
    desc 'Run local unit tests with Docker database
    DB_TRACE - trace SQL queries - default: false
    POSTGRES_VERSION - use specific Postgres database version - optional'
    task :backend_db do
        ENV["STORK_DATABASE_NAME"] = "storktest"
        Rake::Task[:pre_docker_db].invoke()

        at_exit {
            Rake::Task["demo:down"].invoke()
        }

        ENV["DETACH"] = "true"
        Rake::Task["demo:up:postgres"].invoke()
        wait_to_be_operational("postgres")
        Rake::Task["unittest:backend"].invoke()
    end
end

namespace :check do
    desc 'Check the external dependencies related to the demo'
    task :demo do
        check_deps(__FILE__)
    end
end
