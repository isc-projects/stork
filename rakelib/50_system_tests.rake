# System tests
# Run the system tests using docker-compose

#################
### Functions ###
#################

# Displays the hint message with recommended content of the /etc/hosts file
# to handle the Docker hostname resolving correctly.
# Accepts list of all docker-compose files
# Returns False if any hostname is unknown.
def check_hosts_and_print_hint(compose_files)
    require "yaml"

    # List all hostnames of the services that contain Stork Agent.
    # We read the parsed config to resolve the 'extends' statements.
    cmd = [*DOCKER_COMPOSE]
    compose_files.each do |f|
        cmd.append "-f", f
    end
    cmd.append "config"

    stdout, stderr, status = Open3.capture3 *cmd
    if status != 0
        puts stdout, stderr, status
        fail
    end

    hostnames = []
    compose = YAML.load(stdout)
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

#############
### Files ###
#############

# The files generated once by this script
autogenerated = []

# The autogenerated files mounted as volumes
volume_files = []

system_tests_dir = "tests/system"
docker_compose_file_abs = File.expand_path(File.join(system_tests_dir, "docker-compose.yaml"))
kea_many_subnets_dir = "tests/system/config/kea-many-subnets"
directory kea_many_subnets_dir
autogenerated.append kea_many_subnets_dir

kea_many_subnets_config_file = File.join(kea_many_subnets_dir, "kea-dhcp4.conf")
file kea_many_subnets_config_file => [PYTHON, kea_many_subnets_dir] do
    sh PYTHON, "docker/tools/gen_kea_config.py", "7000",
        "-o", kea_many_subnets_config_file,
        "--interface", "eth1"
end
autogenerated.append kea_many_subnets_config_file
volume_files.append kea_many_subnets_config_file

kea_many_subnets_and_shared_networks_config_file = File.join(kea_many_subnets_dir, "kea-dhcp4-sn4400-s13000.conf")
file kea_many_subnets_and_shared_networks_config_file => [PYTHON, kea_many_subnets_dir] do
    sh PYTHON, "docker/tools/gen_kea_config.py", "-n", "4400", "13000",
        "-o", kea_many_subnets_and_shared_networks_config_file,
        "--interface", "eth1"
end
autogenerated.append kea_many_subnets_and_shared_networks_config_file
volume_files.append kea_many_subnets_and_shared_networks_config_file

# These files are generated by the system tests but must exist initially.
lease4_file = "tests/system/config/kea/kea-leases4.csv"
lease6_file = "tests/system/config/kea/kea-leases6.csv"

file lease4_file do
    sh "touch", lease4_file
end

file lease6_file do
    sh "touch", lease6_file
end

CLEAN.append lease4_file, lease6_file
volume_files.append lease4_file, lease6_file

# TLS credentials
tls_dir = "tests/system/config/certs"
cert_file = File.join(tls_dir, "cert.pem")
key_file = File.join(tls_dir, "key.pem")
ca_dir = File.join(tls_dir, "CA")
directory ca_dir

file cert_file => [OPENSSL, ca_dir] do
    sh OPENSSL, "req", "-x509", "-newkey", "rsa:4096",
        "-sha256", "-days", "3650", "-nodes",
        "-keyout", key_file, "-out", cert_file,
        "-subj", "/CN=kea.isc.org", "-addext",
        "subjectAltName=DNS:kea.isc.org,DNS:www.kea.isc.org,IP:127.0.0.1"
end
file key_file => [cert_file]
autogenerated.append cert_file, key_file, ca_dir
volume_files.append cert_file, key_file, ca_dir

# Server API
CLEAN.append *autogenerated

# The system tests log directories
CLEAN.append "test-results/", "tests/system/test-results/", "tests/system/performance-results/"

#########################
### System test tasks ###
#########################

desc 'Run system tests
    TEST - Name of the test to run - optional
    CS_REPO_ACCESS_TOKEN - enables test using the premium Kea hooks - optional
    KEA_VERSION - use specific Kea version - optional
        Supported version formats:
            - MAJOR.MINOR
            - MAJOR.MINOR.PATCH
            - MAJOR.MINOR.PATCH-REVISION
    BIND9_VERSION - use specific BIND9 version - optional, format: MAJOR.MINOR
    POSTGRES_VERSION - use specific Postgres database version - optional
    EXIT_FIRST - exit on the first error - optional, default: false
    ONLY_KEA_TESTS - run only Kea-related tests - optional, default: false'
task :systemtest => [PYTEST, DOCKER_COMPOSE, OPEN_API_GENERATOR_PYTHON_DIR, *GRPC_PYTHON_API_FILES, *volume_files, "systemtest:setup_version_envvars"] do
    opts = []

    # Used in GitLab CI.
    opts.append('--junit-xml=test-results/junit.xml')

    if !ENV["TEST"].nil?
        opts.append "-k", ENV["TEST"]
    end

    if ENV["EXIT_FIRST"] == "true"
        opts.append "--exitfirst"
    end

    # ToDo: Remove the below switches after updating OpenAPI Generator.
    # OpenAPI Generator creates a code that uses the deprecated
    # "HTTPResponse.getheaders()" and "HTTPResponse.getheader()" methods.
    # It causes to generate thousands of warnings during the system tests
    # execution.
    #
    # Full warning message:
    #
    #  /home/deep/Projects/stork/tests/system/openapi_client/rest.py:40: DeprecationWarning: HTTPResponse.getheader() is deprecated and will be removed in urllib3 v2.1.0. Instead use HTTPResponse.headers.get(name, default).
    #    return self.urllib3_response.getheader(name, default)
    #
    #  tests/test_bind9.py::test_bind9
    #  /home/deep/Projects/stork/tests/system/openapi_client/rest.py:36: DeprecationWarning: HTTPResponse.getheaders() is deprecated and will be removed in urllib3 v2.1.0. Instead access HTTPResponse.headers directly.
    #    return self.urllib3_response.getheaders()
    opts.append "-W", "ignore:HTTPResponse.getheaders() is deprecated and will be removed in urllib3 v2.1.0. Instead access HTTPResponse.headers directly.:DeprecationWarning:openapi_client.rest"
    opts.append "-W", "ignore:HTTPResponse.getheader() is deprecated and will be removed in urllib3 v2.1.0. Instead use HTTPResponse.headers.get(name, default).:DeprecationWarning:openapi_client.rest"

    Dir.chdir(system_tests_dir) do
        sh PYTEST, "-s", *opts
    end
end

namespace :systemtest do
    # Sets up the environment variables with Kea and Bind9 versions. Internal task.
    task :setup_version_envvars do
        # Parse Kea version
        if !ENV["KEA_VERSION"].nil?
            kea_version = ENV["KEA_VERSION"]

            # Reject packages for Kea prior to 2.0.0
            kea_eol_major=1
            kea_eol_minor=9

            # Split the version on dash and get the first part. Split it by dots.
            components = kea_version.split('-')[0].split('.')
            if components.length < 2
                fail "You need to specify at least MAJOR.MINOR components of KEA_VERSION variable"
            end
            if components.length >= 3
                kea_version_patch = components[2]
            end
            if components.length > 3
                warn "KEA_VERSION variable contains more than 3 components - ignoring the rest"
            end

            components.each do |c|
                if c.nil?
                    next
                end
                if c == ""
                    fail "KEA_VERSION variable contains an empty component"
                end
                if c !~ /^\d+$/
                    fail "KEA_VERSION variable contains a non-numeric component"
                end
            end

            # Enhance the Kea version with wildcard if the full package is not provided.
            if components.length == 2
                # Add patch wildcard if not provided.
                kea_version += ".*"
            elsif !kea_version.include? '-'
                # Add revision wildcard if the full package name is not provided.
                kea_version += "-*"
            end

            kea_version_info = components.map { |x| x.to_i }
            if kea_version_info.length == 2
                # If the patch is not provided explicitly, the recent patch
                # version is used. We assume it is always bigger than the
                # version thresholds below.
                kea_version_info.append 1000
            end

            # Set the environment variables indicating the versions that
            # changed the package structure.
            if (kea_version_info <=> [kea_eol_major, kea_eol_minor]) <= 0  then
                fail "You need to specify a newer version than #{kea_eol_major}.#{kea_eol_minor} which is EOL."
            end

            kea_prior_2_3_0 = false
            kea_prior_2_7_5 = false

            # Enable legacy packages for Kea prior to 2.3.0
            if (kea_version_info <=> [2, 3]) < 0 then
                kea_prior_2_3_0 = true
                puts "Use the Kea legacy packages prior to 2.3.0"
            end
            if kea_prior_2_3_0
                kea_prior_2_7_5 = true
            elsif (kea_version_info <=> [2, 7, 5]) < 0 then
                kea_prior_2_7_5 = true
                puts "Use the Kea legacy packages prior to 2.7.5"
            end

            # Use single development repository for Kea 2.7.0 and newer.
            ENV["KEA_REPO"] = "isc/kea-#{kea_version_info[0]}-#{kea_version_info[1]}"
            is_development_version = kea_version_info[1] % 2 == 1
            if is_development_version &&
                (kea_version_info <=> [2, 7]) >= 0 then
                ENV["KEA_REPO"] = "isc/kea-dev"
            end

            ENV["KEA_VERSION"] = kea_version
            ENV["KEA_PRIOR_2_3_0"] = kea_prior_2_3_0 ? "true" : "false"
            ENV["KEA_PRIOR_2_7_5"] = kea_prior_2_7_5 ? "true" : "false"
        end
    end

    desc 'List the test cases'
    task :list => [PYTEST, OPEN_API_GENERATOR_PYTHON_DIR, *OPEN_API_GENERATOR_PYTHON_DIR] do
        Dir.chdir(system_tests_dir) do
            sh PYTEST, "--collect-only"
        end
    end

    desc 'Build the containers used in the system tests'
    task :build do
        Rake::Task["systemtest:sh"].invoke("build")
    end

    desc 'Run shell in the docker-compose container
        SERVICE - name of the docker-compose service - required
        SERVICE_USER - user to log in - optional'
    task :shell do
        user = []
        if !ENV["SERVICE_USER"].nil?
            user.append "--user", ENV["SERVICE_USER"]
        end

        Rake::Task["systemtest:sh"].invoke(
            "exec", *user, ENV["SERVICE"], "/bin/sh")
    end

    desc 'Display docker-compose logs
        SERVICE - name of the docker-compose service - optional'
    task :logs do
        service_name = []
        if !ENV["SERVICE"].nil?
            service_name.append ENV["SERVICE"]
        end
        Rake::Task["systemtest:sh"].invoke("logs", *service_name)
    end

    desc 'Run perfdhcp docker-compose service'
    task :perfdhcp do |t, args|
        Rake::Task["systemtest:sh"].invoke("run", "perfdhcp", *args)
    end

    desc 'Run system tests docker-compose
        USE_BUILD_KIT - use BuildKit for faster build - default: true
        CS_REPO_ACCESS_TOKEN - build the containers including Kea premium features - optional
        KEA_VERSION - use specific Kea version - optional
            Supported version formats:
                - MAJOR.MINOR
                - MAJOR.MINOR.PATCH
                - MAJOR.MINOR.PATCH-REVISION
        BIND9_VERSION - use specific BIND9 version - optional, format: MAJOR.MINOR
    '
    task :sh => volume_files + [DOCKER_COMPOSE, :setup_version_envvars] do |t, args|
        if ENV["USE_BUILD_KIT"] != "false"
            ENV["COMPOSE_DOCKER_CLI_BUILD"] = "1"
            ENV["DOCKER_BUILDKIT"] = "1"
        end

        ENV["PWD"] = Dir.pwd

        profiles = []
        if ENV["CS_REPO_ACCESS_TOKEN"]
            puts "Use the Kea premium containers"
            profiles.append "--profile", "premium"
        end

        sh *DOCKER_COMPOSE,
            "-f", docker_compose_file_abs,
            "--project-directory", File.expand_path("."),
            "--project-name", "stork_tests",
            *profiles,
            *args
    end

    desc 'Runs the specific system test docker-compose service. The Stork agent
    will be redirected to the Stork server on the host machine (the server must
    be started separately). Due to the system test framework specific, only one
    service may be running at a time. The service must be shut down before
    starting the system tests.
    This task is dedicated to development purposes and should be used only with
    agent-related services.
        SERVICE - name of the docker-compose service - required'
    task :up do
        service_name = ENV["SERVICE"]
        if service_name.nil?
            fail "You need to specify the SERVICE environment variable"
        end

        if !service_name.start_with? "agent-"
            fail "The task is dedicated for the Stork agent-related services only"
        end

        # System test services related to the same application (Kea or BIND9)
        # share common hostname.
        if !check_hosts_and_print_hint([docker_compose_file_abs])
            fail "Update the /etc/hosts file"
        end

        host_server_address = "http://host.docker.internal:8080"
        if OS == "linux"
            host_server_address = "http://172.42.42.1:8080"
        end
        ENV["STORK_SERVER_URL"] = host_server_address

        Rake::Task["systemtest:sh"].invoke(
            "up", service_name,
            "--abort-on-container-exit"
        )
    end

    desc 'Down all running services, removes networks and volumes'
    task :down do
        Rake::Task["systemtest:sh"].invoke("down", "--volumes", "--remove-orphans")
    end

    desc 'Checks the /etc/hosts file content'
    task :check_etchosts do
        check_hosts_and_print_hint([docker_compose_file_abs])
    end
end

namespace :gen do
    namespace :systemtest do
        desc 'Create autogenerated configs and files'
        task :configs => autogenerated

        desc 'Generate Swagger API files'
        task :swagger => [OPEN_API_GENERATOR_PYTHON_DIR]

        desc 'Generate GRPC API files'
        task :grpc => GRPC_PYTHON_API_FILES
    end
end

namespace :prepare do
    desc 'Install the external dependencies related to the system tests'
    task :systemtest do
        find_and_prepare_deps(__FILE__)
    end
end

namespace :check do
    desc 'Check the external dependencies related to the system tests'
    task :systemtest do
        check_deps(__FILE__)
    end
end
