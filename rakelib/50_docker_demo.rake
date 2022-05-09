# Demo
# Run the demo containers in Docker

namespace :demo do
    
    #################
    ### Functions ###
    #################

    # Displays the hint message with recommended content of the /etc/hosts file
    # to handle the Docker hostname resolving correctly.
    # Returns False if any hostname is unknown.
    def check_hosts_and_print_hint()
        require "yaml"
        # List of all docker-compose files
        compose_files = FileList["docker/docker-compose*.yaml"]
        # List all hostnames of the services that contain Stork Agent.
        hostnames = []
        compose_files.each do |f|
            compose = YAML.load_file(f)
            compose["services"].each do |name, service|
                # Ignore non-agent services
                if !name.start_with? "agent"
                    next
                end
                # Default hostname
                hostname = name
                # Custom hostname
                if service.key? "hostname"
                    hostname = service["hostname"]
                end
                hostnames.append hostname
            end
        end

        # List all unknown hostnames
        unknown_hostnames = []
        hostnames.each do |h|
            _, _, status = Open3.capture3 "ping", "-c", "1", h
            if status != 0
                unknown_hostnames.append h
            end
        end

        # Print message
        if !unknown_hostnames.empty?
            puts "Some Docker hostnames cannot be resolved."
            puts "You need to append the below entries to your /etc/hosts file."
            puts "They redirect to localhost because the main docker-compose network uses the bridge mode."
            puts "--- Start /etc/hosts content ---"
            unknown_hostnames.each do |h|
                print "127.0.0.1", "\t", h, "\n"
            end
            puts "--- End /etc/hosts content ---"
        end

        # OK - all hostnames are known
        return unknown_hostnames.empty?
    end
    
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
    # Environment variables:
    # CS_REPO_ACCESS_TOKEN - CloudSmith repo token, required for premium services
    def get_docker_opts(server_mode, cache, services)
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
        
        up_opts = [
            "--attach-dependencies",
            "--remove-orphans",
            # Warning! Don't use here "--renew-anon-volumes" options. It causes conflicts between running containers. 
        ]
        additional_services = []
        
        if server_mode == "host"
            if !check_hosts_and_print_hint()
                fail "Update the /etc/hosts file"
            end
            host_server_address = "http://host.docker.internal:8080"
            if OS == "linux"
                host_server_address = "http://172.20.0.1:8080"
            end
            ENV["STORK_SERVER_URL"] = host_server_address
            up_opts += ["--scale", "server=0", "--scale", "webui=0"]
        elsif server_mode == "with-ui"
            if !services.empty?
                additional_services.append "webui"
            end
        elsif server_mode == "without-ui"
            if !services.empty?
                additional_services.append "server"
            end
            up_opts += ["--scale", "webui=0"]
        elsif server_mode == "no-server"
            up_opts += ["--scale", "server=0", "--scale", "webui=0"]
            # Prevents the Stork Agent from the registration
            ENV["STORK_SERVER_MODE_URL"] = ""
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
    def docker_up_services(*services)
        # Read arguments from the environment variables
        server_mode = ENV["SERVER_MODE"]
        cache = ENV["CACHE"] != "false"
        
        # Prepare the docker-compose flags
        opts, build_opts, up_opts, additional_services = get_docker_opts(server_mode, cache, services)
        
        # We don't use the BuildKit features in our Dockerfiles (yet).
        # But we turn on the BuildKit to build the Docker stages concurrently and skip unnecessary stages.  
        ENV["COMPOSE_DOCKER_CLI_BUILD"] = "1"
        ENV["DOCKER_BUILDKIT"] = "1"
        
        # Execute the docker-compose commands
        sh "docker-compose", *opts, "build", *build_opts, *services, *additional_services
        sh "docker-compose", *opts, "up", *up_opts, *services, *additional_services
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
    task :up do
        docker_up_services()
    end
    
    namespace :up do
        desc 'Build and run container with Stork Agent and Kea
        See "up" command for arguments.'
        task :kea do
            docker_up_services("agent-kea")
        end
        
        desc 'Build and run container with Stork Agent and Kea with many subnets in the configuration.
        See "up" command for arguments.'
        task :kea_many_subnets do
            docker_up_services("agent-kea-many-subnets")
        end
        
        desc 'Build and run container with Stork Agent and Kea DHCPv6 server
        See "up" command for arguments.'
        task :kea6 do
            docker_up_services("agent-kea6")
        end
        
        desc 'Build and run two containers with Stork Agent and Kea HA pair
        See "up" command for arguments.'
        task :kea_ha do
            docker_up_services("agent-kea-ha1", "agent-kea-ha2")
        end
        
        desc 'Build and run container with Stork Agent and Kea with host reseverations in db
        CS_REPO_ACCESS_TOKEN - CloudSmith token - required
        See "up" command for more arguments.'
        task :kea_premium do
            if !ENV["CS_REPO_ACCESS_TOKEN"]
                fail 'You need to provide the CloudSmith access token in CS_REPO_ACCESS_TOKEN environment variable.'
            end
            docker_up_services("agent-kea-premium")
        end
        
        desc 'Build and run container with Stork Agent and BIND 9
        See "up" command for arguments.'
        task :bind9 do
            docker_up_services("agent-bind9")
        end
        
        desc 'Build and run container with Postgres'
        task :postgres do
            docker_up_services("postgres")
        end
        
        desc 'Build and run Docker DNS Proxy Server to resolve internal Docker hostnames'
        # Source: https://stackoverflow.com/a/45071285
        task :dns_proxy_server do
            docker_up_services("dns-proxy-server")
        end
    end
    
    desc 'Down all containers and remove all volumes'
    task :down do
        ENV["CS_REPO_ACCESS_TOKEN"] = "stub"
        opts, _, _, _ = get_docker_opts(nil, false, [])
        sh "docker-compose", *opts, "down",
        "--volumes",
        "--remove-orphans",
        "--rmi", "local"
    end

    desc 'Checks the /etc/hosts file content'
    task :check_etchosts do
        check_hosts_and_print_hint()
    end
    
    #######################
    ### Docker registry ###
    #######################
    
    # ToDo: This task is not refactored.
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
# DB_TRACE - trace SQL queries - default: false
task :pre_docker_db do
    if ENV["DB_TRACE"] == "true"
        ENV["STORK_DATABASE_TRACE"] = "run"
    end
    
    ENV["STORK_DATABASE_HOST"] = "172.20.0.234"
    ENV["STORK_DATABASE_PORT"] = "5432"
    ENV["STORK_DATABASE_USER_NAME"] = "stork"
    ENV["STORK_DATABASE_PASSWORD"] = "stork"
    ENV["STORK_DATABASE_NAME"] = "stork"
    ENV['PGPASSWORD'] = "stork"
end

namespace :run do
    desc 'Run local server with Docker database
    DB_TRACE - trace SQL queries - default: false'
    task :server_db => [:pre_docker_db] do |t|
        Rake::MultiTask.new(:stub, t.application)
        .enhance(["run:server", "demo:up:postgres"])
        .invoke()
    end
end

namespace :unittest do
    desc 'Run local unit tests with Docker database
    DB_TRACE - trace SQL queries - default: false'
    task :backend_db => [:pre_docker_db] do |t|
        Rake::MultiTask.new(:stub, t.application)
        .enhance(["unittest:backend", "demo:up:postgres"])
        .invoke()
    end
end
