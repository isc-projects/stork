# Utilities
# The file contains helpful tasks
# for developers that aren't strictly
# related to the source code.

namespace :utils do
    desc 'Generate ctags for Emacs'
    task :ctags => [ETAGS_CTAGS] do
        sh ETAGS_CTAGS,
        "-f", "TAGS",
        "-R",
        "--exclude=webui/node_modules",
        "--exclude=webui/dist",
        "--exclude=tools",
        "."
    end


    desc 'Connect gdlv GUI Go debugger to waiting dlv debugger'
    task :connect_dbg => [GDLV] do
        sh GDLV, "connect", "127.0.0.1:45678"
    end

    desc "List dependencies of a given package
        Choose one:
            ABS - full absolute package import path
            REL - package path relative to main Stork directory"
    task :list_package_deps => [GO] do
        Dir.chdir "backend" do
            if !ENV["ABS"].nil?
                package = ENV["ABS"]
            elsif !ENV["REL"].nil?
                if ENV["REL"].start_with? "backend/"
                    ENV["REL"] = ENV["REL"].delete_prefix "backend/"
                end

                package = File.join("isc.org/stork", ENV["REL"])
            else
                fail "You need to provide the ABS or REL variable"
            end

            stdout, _ = Open3.capture2 GO, "list", "-f", '# Package - import: {{ .ImportPath }} name: {{ .Name }}', package
            puts stdout

            stdout, _ = Open3.capture2 GO, "list", "-f", '{{ join .Deps "\n" }}', package

            std_deps = []
            external_deps = []

            stdout.split("\n").each do |d|
                stdout, _ = Open3.capture2 GO, "list", "-f", '{{ .Standard }}', d
                if stdout.strip == "true"
                    std_deps.append d
                else
                    external_deps.append d
                end
            end

            puts "# Dependency packages from standard library"
            std_deps.each do |d|
                puts d
            end

            puts
            puts "# External dependency packages"
            external_deps.each do |d|
                puts d
            end
        end
    end

    desc "List platforms supported by the installed Go version"
    task :list_go_supported_platforms => [GO] do
        sh GO, "tool", "dist", "list"
    end

    desc 'List packages in a given Docker file and prints the newest available versions
        DOCKERFILE - path to the Dockerfile - required'
    task :list_packages_in_dockerfile => DOCKER do
        dockerfile = ENV["DOCKERFILE"]
        if dockerfile.nil?
            fail "You must specify the path to the Dockerfile: DOCKERFILE=/path/to/Dockerfile"
        end

        # Key is the package manager install command.
        # Value is an array of two elements:
        #   - package manager update command
        #   - lambda that returns the command to check the package version
        #   - the package name and version delimiter
        package_managers = {
            "apt-get install" => [
                ["apt-get", "update"],
                -> (name) {["/bin/sh", "-c", "apt-cache madison #{name} | head -n 1 | cut -d'|' -f2"]},
                "="
            ],
            "yum install" => [
                ["yum", "updateinfo"],
                -> (name) {["/bin/sh", "-c", "yum info #{name} | grep Version | head -n 1 | cut -d':' -f2"]},
                "-"
            ],
            "dnf install" => [
                ["dnf", "updateinfo"],
                -> (name) {["/bin/sh", "-c", "dnf info #{name} | grep Version | head -n 1 | cut -d':' -f2"]},
                "-"
            ],
            "apk add" => [
                ["apk", "update"],
                -> (name) {["/bin/sh", "-c", "apk info #{name} | head -n 1 | sed -e 's/^#{name}-\\(.*\\) .*$/\\1/'"]},
                "~"
            ]
        }

        # The key of the currently processed package manager call.
        # If the parsing loop is not in the package manager call, the value is
        # nil.
        package_manager_key = nil
        # The base image of the currently processed stage. If the stage is
        # built on top of another (using the FROM ... AS ... syntax), it
        # contains the name of the stage (after the AS keyword). Otherwise, it
        # contains the name of the base image.
        base_image = nil
        # The name of the Docker container for fetching the dependencies for
        # the currently processed stage. It is created from the base image.
        container_name = nil
        # The names of the already used containers. They will be removed at the
        # end of the script.
        container_names = []

        # The dictionary of the relations between the stages. The key is the
        # stage name (spefied after the AS keyword) and the value is the name
        # of the parent stage (specified after the FROM keyword).
        # The dictionary is used to find the most precedent base image.
        stage_parents = { }
        # The dictionary of the arguments (ARG) and environment variables (ENV)
        # specified in the Dockerfile. It is used to substitute the variables.
        arguments_and_envs = { }
        # The list of the detected packages. Each element is an array of the
        # following elements:
        #   - base image
        #   - package name
        #   - current version (version specified in the Dockerfile)
        #   - latest version (latest version available in the base image)
        #   - boolean value indicating if the package is up-to-date
        packages = []

        File.open(dockerfile, "r") do |f|
            f.each_line do |line|
                # Split the content and comment.
                parts = line.split("#", 2)
                line_content = parts[0]

                # Strip the line.
                line_content = line_content.strip

                # Substitute the environment variables.
                line_content = line_content.gsub(/\$\{([a-zA-Z0-9_]+)\}/) do |match|
                    # TODO: It may be useful to support overriding the values
                    # with the environment variables.
                    if !arguments_and_envs[$1].nil?
                        next arguments_and_envs[$1]
                    else
                        fail "The argument or environment variable #{$1} is not defined"
                    end
                end

                # Skip empty lines.
                if line_content.empty?
                    next
                end

                # Check if the line is a beginning of a new stage.
                if line_content =~ /^FROM\s+(?:--platform=.*?\s+)?(.*)$/i
                    base_image = $1
                    # Check if it is a stage.
                    if base_image =~ /(.*)\s+AS\s+(.*)/i
                        # It is a stage. Split the parent image and the stage
                        # name. Remember the relation between the stage and its
                        # parent. Set the base image to the stage name.
                        parent = $1
                        child = $2
                        stage_parents[child] = parent
                        base_image = child
                    end
                # Check if the line is a declaration of an argument or an
                # environment variable. Support the delimiting the key and the
                # value with the equal sign (ARG key=value) or the space
                # (ENV key value).
                elsif line_content =~ /^(?:ARG|ENV)\s+(.*)(?:=|\s+)(.*)$/i
                    value = $2
                    # Strip the quotes.
                    if value.start_with? '"' and value.end_with? '"'
                        value = value[1..-2]
                    end
                    arguments_and_envs[$1] = $2
                elsif !package_manager_key.nil?
                    # We are in the package manager call.

                    # Check if the line is a beginning of another command.
                    if line_content.start_with? "&&" or line_content.start_with? "||" or line_content.start_with? ";"
                        package_manager_key = nil
                        next
                    end

                    # Skip the flags.
                    if line_content.start_with? "-"
                        next
                    end

                    # Check if line is last. The line is last if it doesn't end
                    # with the breaking line character (\)
                    is_last_line = false
                    if !line_content.end_with? '\\'
                        # End line.
                        is_last_line = true
                    else
                        # Strip the trailing backslash.
                        line_content = line_content[0..-2]
                        line_content = line_content.strip
                    end

                    # Extract the package manager-specific information.
                    create_check_command = package_managers[package_manager_key][1]
                    version_delimiter = package_managers[package_manager_key][2]

                    # Split the version if any.
                    package_name, _, current_version = line_content.rpartition(version_delimiter)
                    if package_name == ""
                        # Has no version.
                        package_name = current_version
                        current_version = "unspecified"
                    elsif current_version.end_with? "*"
                        # Has a version with a wildcard.
                        # Strip the trailing asterisk.
                        current_version = current_version[0..-2]
                    end

                    # Check the available version in the base image.
                    check_command = create_check_command.call(package_name)
                    stdout, stderr, status = Open3.capture3 DOCKER, "exec", container_name, *check_command
                    if status != 0
                        fail "Failed to check the package version, status: #{status}, stderr: #{stderr}, stdout: #{stdout}"
                    end
                    stdout = stdout.strip
                    available_version = stdout

                    # The current version is up-to-date if it prefixes the
                    # available version. We support only the equality (or
                    # semi-equality) operators.
                    is_up_to_date = available_version.start_with? current_version

                    # Save the result.
                    packages.append [base_image, package_name, current_version, available_version, is_up_to_date.to_s]

                    if is_last_line
                        package_manager_key = nil
                    end

                    next
                end

                # Check if the line is a package manager call.
                # We expect that the package manager call will not contain any
                # package names.
                package_managers.each do |key, _|
                    if line_content =~ /#{key}/
                        package_manager_key = key
                        break
                    end
                end

                # Check if any package manager was found.
                if !package_manager_key.nil?
                    # Pattern to sanitize the container name. It matches the characters
                    # accepted by Docker.
                    sanity_pattern = /[^a-zA-Z0-9_.-]/

                    # Search for the most precedent base image.
                    parent_image = base_image
                    while !stage_parents[parent_image].nil?
                        parent_image = stage_parents[parent_image]
                    end

                    # Start a container.
                    container_name = "stork-#{parent_image}-#{package_manager_key}".gsub(sanity_pattern, "_")

                    # If the container was already used, it is not necessary to create it again.
                    if container_names.include? container_name
                        next
                    end

                    # Add the container to the list of used containers to
                    # clean up them later.
                    container_names.append container_name

                    # Remove the container if it exists.
                    sh DOCKER, "rm", "-f", container_name
                    # Create and run the container.
                    sh DOCKER, "run", "-d", "--name", container_name, parent_image, "sleep", "infinity"
                    # Update the package manager database.
                    package_update_command = package_managers[package_manager_key][0]
                    sh DOCKER, "exec", container_name, *package_update_command
                end
            end
        end

        # Clean up used containers.
        container_names.each do |container_name|
            # Stop the container.
            sh DOCKER, "stop", container_name
            # Remove the container.
            sh DOCKER, "rm", "-f", container_name
        end

        # Print the result.
        line_format = "%-40s %-40s %-30s %-40s %-10s\n"
        printf line_format, "Base image", "Package name", "Current version", "Latest version", "Up-to-date"
        packages.each do |p|
            printf line_format, *p
        end
    end
end


namespace :prepare do
    desc 'Install the external dependencies related to the codebase'
    task :utils do
        find_and_prepare_deps(__FILE__)
    end
end


namespace :check do
    desc 'Check the external dependencies related to the utils'
    task :utils do
        check_deps(__FILE__)
    end
end
