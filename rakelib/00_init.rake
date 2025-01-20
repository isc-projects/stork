# coding: utf-8

# Initialization
# This file contains the toolkits that
# aren't related to the source code.
# It means that they don't change very often
# and can be cached for later use.

require 'digest'
require 'open3'

# Cross-platform way of finding an executable in the $PATH.
# Source: https://stackoverflow.com/a/5471032
#
#   which('ruby') #=> /usr/bin/ruby
def which(cmd)
    if File.executable?(cmd)
        return cmd
    end

    exts = ENV['PATHEXT'] ? ENV['PATHEXT'].split(';') : ['']
    ENV['PATH'].split(File::PATH_SEPARATOR).each do |path|
      exts.each do |ext|
        exe = File.join(path, "#{cmd}#{ext}")
        return exe if File.executable?(exe) && !File.directory?(exe)
      end
    end
    nil
end

# Returns true if the libc-musl variant of the libc library is used. Otherwise,
# returns false (the standard variant is used).
def detect_libc_musl()
    platform = Gem::Platform.local
    if platform.version.nil?
        return false
    end
    return platform.version == "musl"
end

# Indicates if the provided task should be considered as a dependency.
# The dependencies are all file tasks and some particular standard tasks that
# define the custom logic to set up and check the dependency. They are
# recognized by the existing :@manual_install variable.
def is_dependency_task(t)
    t.class == Rake::FileTask || !t.instance_variable_get(:@manual_install).nil?
end

# Searches for the tasks in the provided file
def find_tasks(file)
    tasks = []
    # Iterate over all tasks
    Rake.application.tasks.each do |t|
        # Choose only tasks from a specific file
        if t.actions.empty?
            next
        end
        action = t.actions[0]
        location, _ = action.source_location
        if location != file
            next
        end
        tasks.append t
    end
    return tasks
end

# Searches for the prerequisites tasks from the second provided file in the
# first provided file.
def find_prerequisites_tasks(source_tasks_file, prerequisites_file)
    require 'set'
    unique_prerequisites = Set[]

    # Choose only tasks from a specific file
    tasks = find_tasks(source_tasks_file)

    # Iterate over tasks
    tasks.each do |t|
        # Iterate over prerequisites
        t.all_prerequisite_tasks.each do |p|
            # Select unique prerequisites
            unique_prerequisites.add p
        end
    end

    prerequisites_tasks = []

    # Check the prerequisites
    unique_prerequisites.each do |p|
        # Check the location - accept only tasks from the init file
        if p.actions.empty?
            next
        end

        action = p.actions[0]
        location, _ = action.source_location

        if location == prerequisites_file
            prerequisites_tasks.append p
        end
    end

    return prerequisites_tasks
end

# Searches for the prerequisites from the init file in the provided file and
# invoke them.
def find_and_prepare_deps(file)
    if file == __FILE__
        prerequisites_tasks = find_tasks(file)
    else
        prerequisites_tasks = find_prerequisites_tasks(file, __FILE__)
    end

    prerequisites_tasks.each do |t|
        if !is_dependency_task(t)
            next
        end

        if t.instance_variable_get(:@manual_install)
            # Skips the missing top-level manually-installed prerequisites
            # to avoid interrupting preparing operation. If the
            # manually-installed prerequisite is a dependency of any
            # prerequisites and it's missing, then the preparing operation is
            # stopped anyway.
            if which(t.to_s).nil?
                puts "Preparing: #{t.name}... must be manually installed"
                next
            end
        end

        print "Preparing: ", t, "...\n"
        t.invoke()
    end
end

# Searches for the prerequisites from the init file in the provided file and
# checks if they exist.
def check_deps(file)
    def print_status(name, path, ok)
        status = "[ OK ]"
        if !ok
            status = "[MISS]"
        end

        if !path.nil?
            path = " (" + path + ")"
        end

        print status, " ", name, path, "\n"
    end

    if file == __FILE__
        prerequisites_tasks = find_tasks(file)
    else
        prerequisites_tasks = find_prerequisites_tasks(file, __FILE__)
    end

    manual_install_prerequisites_tasks = []
    prerequisites_tasks.each do |t|
        if t.instance_variable_get(:@manual_install)
            manual_install_prerequisites_tasks.append t
        end
    end

    manual_install_prerequisites_tasks.each do |t|
        prerequisites_tasks.delete t
    end

    puts "Self-installed dependencies:"
    prerequisites_tasks.sort_by{ |t| t.name().rpartition("/")[2] }.each do |t|
        if !is_dependency_task(t)
            next
        end

        path = t.to_s
        name = path
        _, _, name = path.rpartition("/")

        print_status(name, path, !t.needed?)
    end

    puts "\nManually-installed dependencies:"

    manual_install_prerequisites_tasks
        .map { |p| [p.name().rpartition("/")[2], p.name(), !p.needed? ] }
        .sort_by{ |name, _, _| name }
        .each { |args| print_status(*args) }
end

# General-purpose guard for file tasks. It generates an empty file with the
# name composed of task name, arbitrary identifier, and suffix. The file is
# appended to the task prerequisites list.
# The guarded file task will be re-executed if the identifier is updated.
# It allows file tasks to depend on non-standard conditions.
def add_guard(task_name, identifier, suffix)
    task = Rake::Task[task_name]
    if task.class != Rake::FileTask
        fail "file task required"
    end

    # We don't use the guard for the prerequisities that must be
    # installed manually on current operating system
    if task.instance_variable_get(:@manual_install)
        return
    end

    # The stamp file is a prerequisite, but it is created after the
    # guarded task. It allows for cleaning the target directory in the task
    # body.
    stamp = "#{task_name}-#{identifier}.#{suffix}"
    file stamp
    task.enhance [stamp] do
        # Removes old stamps
        FileList["#{task_name}-*.#{suffix}"].each do |f|
            FileUtils.rm f
        end
        # Creates a new stamp with a timestamp before the guarded task
        # execution.
        FileUtils.touch [stamp], mtime: task.timestamp
    end
end

# Defines the version guard for a file task. The version guard allows file
# tasks to depend on the version from the Rake variable. Using it for the tasks
# that have frozen versions using external files is not necessary.
# It accepts a task to be guarded and the version.
def add_version_guard(task_name, version)
    add_guard(task_name, version, "version")
end

# Defines the hash guard for a file task. The hash guard allows file tasks to
# depend on the hash file instead of its timestamp. It prevents re-executing
# tasks that depend on the external files while their dependencies are
# refreshed but not changed (e.g., the repository is re-cloned).
# It accepts a task to be guarded and the dependency file.
# The dependency file should not be included in the prerequite list of the task.
def add_hash_guard(task_name, prerequisite_file)
    hash = Digest::SHA256.file(prerequisite_file).hexdigest
    add_guard(task_name, hash, "hash")
end


# Defines a file task with no logic and always has the "not needed" status.
# The file task is rebuilt if the task target is updated (the modification date
# is later than remembered by Rake) or if any prerequisites are updated. The
# task that was changed has the "needed" status in Rake. The tasks created
# using this function are always up-to-date and don't trigger the rebuild of
# the parent task. They always have a "not needed" status and timestamp earlier
# than the parent task.
def create_not_needed_file_task(task_name)
    file task_name

    Rake::Task[task_name].tap do |task|
        def task.timestamp # :nodoc:
            Time.at 0
        end

        def task.needed?
            false
        end
    end

    return task_name
end

# This is a regular task that does nothing. It is dedicated to using as
# prerequirement. Due to it being a regular task it is always recognized as
# "needed" and causes to rebuild of a parent task.
# The file tasks with this prerequisite cannot be a prerequisite for other file
# tasks to avoid a negative impact on the performance.
#
# This task works similarly to a default "phony" task in Ruby but has
# additional validation that prevents you from using it in the middle of the
# dependency chain. It isn't necessary; it changes nothing in how this task
# works, but it verifies if a developer didn't misuse it and provide a
# hard-to-find bug. See discussion about the "phony" task in
# https://gitlab.isc.org/isc-projects/stork/-/merge_requests/535#note_344019.
task :always_rebuild_this_task do |this|
    # Checks if no file task depends on a file task with this prerequisite.
    Rake::Task.tasks().each do |t|
        if !is_dependency_task(t)
            next
        end

        # Iterates over the prerequisities of the file task.
        t.prerequisites.each do |p|
            # Iterates over the nested prerequisities (prerequisities of
            # prerequisities).
            Rake::Task[p].all_prerequisite_tasks.each do |n|
                if n == this
                    fail "#{this} cannot be a prerequsite of file task (#{t})"
                end
            end
        end
    end
end

# Create a new file task that fails if the executable doesn't exist.
# It accepts a path to the executable.
def create_manually_installed_file_task(path)
    file path do
        # This check allows to use manually installed tasks as prerequisities of
        # other manually installed tasks.
        if which(path).nil?
            fail "#{path} must be installed manually on your operating system"
        end
    end

    # Add a property to indicate that it's a manually installed file.
    newTask = Rake::Task[path]
    newTask.instance_variable_set(:@manual_install, true)
    return newTask.name
end

# Task name should be a path to file or an executable name.
#
# If the path is used, it must be a name of the existing path. If all conditions
# are false, the function leaves the task and task name untouched.
#
# If the task name is a name of the executable (no slashes) then it may not
# have related file task. If all conditions are false, the function creates
# a dump/phony task and leaves the task name untouched.
#
# If any condition is true, the original task is removed. It is replaced with
# a new one that fails if the task target doesn't exist. The task has a property
# that indicates that it requires a manual install. Function returns a new name
# that depends on the which command output.
def require_manual_install_on(task_name, *conditions)
    task = nil
    if Rake::Task.task_defined? task_name
        task = Rake::Task[task_name]
    end

    # The task may not exist for the executables that must be found in PATH.
    # Other files must have assigned file tasks.
    if (!task.nil? && task.class != Rake::FileTask) || (task.nil? && task_name.include?("/"))
        fail "file task required"
    end

    if !conditions.any?
        if task.nil?
            # Create an empty file task to prevent failure due to a non-existing
            # file if the executable isn't prerequisite.
            create_not_needed_file_task(task_name)
        end
        return task_name
    end

    # Remove the self-installed task when it is unsupported.
    if !task.nil?
        task.clear()
        Rake.application.instance_variable_get('@tasks').delete(task_name)
    end

    # Search in PATH for executable.
    program = File.basename task_name
    system_path = which(program)
    if !system_path.nil?
        program = system_path
    end

    # Create a new task that fails if the executable doesn't exist.
    return create_manually_installed_file_task(program)
end

# Creates a prerequirement task for the Docker plugin.
# Accepts the name of the standalone executable name (for old Docker version)
# and Docker subcommand name (for modern Docker versions).
# The function picks the available way to execute the plugin.
# The command takes precedence over the standalone executable.
# The created prerequisite should be used with the splat operator in the 'sh'
# calls.
def docker_plugin(standalone_exe, command_name)
    # Docker compose requirement task
    task_name = standalone_exe
    task task_name => [DOCKER] do
        # The docker plugin must be manually installed. If it is installed,
        # the task body is never called.
        fail "docker #{command_name} plugin or #{standalone_exe} standalone is not installed"
    end

    plugin_task = Rake::Task[task_name]
    plugin_task.tap do |task|
        # The non-file tasks with the manual_install variable are considered as
        # dependencies. Set to true to mark it must be manually installed.
        task.instance_variable_set(:@manual_install, true)

        # The functions or methods defined in other functions don't have
        # an access to the variables from the outer scope in Ruby.
        task.instance_variable_set(:@standalone_exe, standalone_exe)
        task.instance_variable_set(:@command_name, command_name)
        task.instance_variable_set(:@task_name, task_name)

        def task.standalone_exe()
            self.instance_variable_get(:@standalone_exe)
        end

        def task.command_name()
            self.instance_variable_get(:@command_name)
        end

        # Check if the docker plugin is installed.
        def is_docker_plugin_command_supported()
            begin
                _, _, status = Open3.capture3 DOCKER, command_name
                return status == 0
            rescue
                # Missing docker command in system.
                return false
            end
        end

        # Check if the standalone executable is installed.
        def task.is_docker_plugin_standalone_supported()
            return !which(standalone_exe).nil?
        end

        # Check if the task should be called. It is internally called by Rake.
        # Return false if the docker plugin is ready to use.
        def task.needed?
            # Fail the task if the docker plugin is missing.
            return !is_docker_plugin_command_supported() && !is_docker_plugin_standalone_supported()
        end

        # Return the string representation of the task.
        def task.name
            if is_docker_plugin_command_supported() || !is_docker_plugin_standalone_supported()
                return "#{DOCKER} #{command_name}"
            else
                return which(standalone_exe)
            end
        end

        # Return the task name. It is used for compatibility with Rake. The
        # identifier of the task and the to_s output must be the same.
        def task.to_s
            self.instance_variable_get(:@task_name)
        end

        # Handle the splat operator call (*task_name). The splat operator should
        # be used to call the task-related command.
        # E.g.: sh *DOCKER_COMPOSE, --foo, --bar
        def task.to_a
            if is_docker_plugin_command_supported() || !is_docker_plugin_standalone_supported()
                return [DOCKER, command_name]
            else
                return [which(standalone_exe)]
            end
        end
    end
    return plugin_task
end

# Fetches the file from the network. You should add the WGET to the
# prerequisites of the task that uses this function.
# The file is saved in the target location.
def fetch_file(url, target)
    # extract wget version
    stdout, _, status = Open3.capture3(WGET, "--version")
    wget = [WGET]

    # BusyBox edition has no version switch and supports only basic features.
    if status == 0
        wget.append "--tries=inf", "--waitretry=3"
        wget_version = stdout.split("\n")[0]
        wget_version = wget_version[/[0-9]+\.[0-9]+/]
        # versions prior to 1.19 lack support for --retry-on-http-error
        if wget_version.empty? or wget_version >= "1.19"
            wget.append "--retry-on-http-error=429,500,503,504"
        end
    end

    if ENV["CI"] == "true"
        # Suppress verbose output on the CI.
        wget.append "--no-verbose"
    end

    wget.append url
    wget.append "-O", target

    sh *wget
end

### Recognize the operating system
uname_os, _ = Open3.capture2 "uname", "-s"
case uname_os.rstrip
    when "Darwin"
        OS="macos"
    when "Linux"
        OS="linux"
    when "FreeBSD"
        OS="FreeBSD"
    when "OpenBSD"
        OS="OpenBSD"
    else
        puts "ERROR: Unknown/unsupported OS: #{uname_os}"
        fail
end

uname_arch, _ = Open3.capture2 "uname", "-m"
case uname_arch.rstrip
    when "x86_64", "amd64"
        ARCH="amd64"
    when "aarch64_be", "aarch64", "armv8b", "armv8l", "arm64"
        ARCH="arm64"
    else
        puts "ERROR: Unknown/unsupported architecture: #{uname_arch}"
        fail
end

if !ENV["GOOS"].nil? || !ENV["GOARCH"].nil? || !ENV["GOARM"].nil?
    puts "You shouldn't use the GOOS, GOARCH, or GOARM environment variables to build Stork."
    puts "Some development toolkits that are needed to build Stork are written in Go."
    puts "They run on your local environment, so they must be compiled for the current operating system and architecture."
    puts "Using the above variables would break these toolkits."
    puts "To build the Stork binaries for a specific operating system or architecture,"
    puts "use the STORK_GOOS, STORK_GOARCH, and (optionally) STORK_GOARM environment variables."
    puts "They accept the same values as the original GOOS, GOARCH, and GOARM."
    fail
end

### Tasks support conditions
# Some prerequisites are related to the libc library but
# without official libc-musl variants. They cannot be installed using this Rake
# script.
libc_musl_system = detect_libc_musl()
# Some prerequisites doesn't have a public packages for BSD-like operating
# systems.
freebsd_system = OS == "FreeBSD"
openbsd_system = OS == "OpenBSD"
arm64_system = ARCH == "arm64"
freebsd_arm64_system = freebsd_system && arm64_system
macos_arm64_system = OS == "macos" && arm64_system
any_system = true

### Define package versions
go_ver = '1.23.5'
goswagger_ver = 'v0.31.0'
protoc_ver = '29.3'
protoc_gen_go_ver = 'v1.36.3'
protoc_gen_go_grpc_ver = 'v1.5.1'
tparse_ver = 'v0.16.0'
go_junit_report_ver = 'v2.1.0'
gocover_cobertura_ver = 'v1.3.0'
go_live_pprof_ver = 'v1.0.8'
govulncheck_ver = 'v1.1.4'
mockgen_ver = 'v0.5.0'
dlv_ver = 'v1.24.0'
gdlv_ver = 'v1.13.1'
nfpm_ver = 'v2.41.2'
golangcilint_ver = '1.63.4'
node_ver = '20.17.0'
npm_ver = '10.9.2'
yamlinc_ver = '0.1.10'
storybook_ver = '8.5.0'
openapi_generator_ver = '7.10.0'
bundler_ver = '2.5.19'
shellcheck_ver = '0.10.0'
pip_tools_ver = '7.4.1'
pip_audit_ver = '2.7.3'

# System-dependent variables
case OS
when "macos"
    case ARCH
    when "amd64"
        go_suffix = "darwin-amd64"
        protoc_suffix = "osx-x86_64"
        node_suffix = "darwin-x64"
        golangcilint_suffix = "darwin-amd64"
        goswagger_suffix = "darwin_amd64"
        shellcheck_suffix = "darwin.x86_64"
    when "arm64"
        go_suffix = "darwin-arm64"
        protoc_suffix = "osx-aarch_64"
        node_suffix = "darwin-arm64"
        golangcilint_suffix = "darwin-arm64"
        goswagger_suffix = "darwin_arm64"
        # Shellcheck has no binaries for Darwin ARM: https://github.com/koalaman/shellcheck/issues/2714
    end
    puts "WARNING: MacOS is not officially supported, the provisions for building on MacOS are made"
    puts "WARNING: for the developers' convenience only."
when "linux"
    case ARCH
    when "amd64"
        go_suffix = "linux-amd64"
        protoc_suffix = "linux-x86_64"
        node_suffix = "linux-x64"
        golangcilint_suffix = "linux-amd64"
        goswagger_suffix = "linux_amd64"
        shellcheck_suffix = "linux.x86_64"
    when "arm64"
        go_suffix = "linux-arm64"
        protoc_suffix = "linux-aarch_64"
        node_suffix = "linux-arm64"
        golangcilint_suffix = "linux-arm64"
        goswagger_suffix = "linux_arm64"
        shellcheck_suffix = "linux.aarch64"
    end
when "FreeBSD"
    case ARCH
    when "amd64"
         go_suffix = "freebsd-amd64"
         golangcilint_suffix = "freebsd-amd64"
    when "arm64"
        golangcilint_suffix = "freebsd-armv7"
    end
when "OpenBSD"
else
  puts "ERROR: Unknown/unsupported OS: %s" % uname_os
  fail
end

### Define dependencies

# Directories
tools_dir = File.expand_path('tools')
directory tools_dir

node_dir = File.join(tools_dir, "nodejs")
directory node_dir

go_tools_dir = File.join(tools_dir, "golang")
gopath = File.join(go_tools_dir, "gopath")
directory go_tools_dir
directory gopath
file go_tools_dir => [gopath]

ruby_tools_dir = File.join(tools_dir, "ruby")
directory ruby_tools_dir

# We use the "bundle" gem to manage the dependencies. The "bundle" package is
# installed using the "gem" executable in the tools/ruby/gems directory, and
# the link is created in the tools/ruby/bin directory. Next, Ruby dependencies
# are installed using the "bundle". It creates the tools/ruby/ruby/[VERSION]/
# directory with "bin" and "gems" subdirectories and uses these directories as
# the location of the installations. We want to avoid using a variadic Ruby
# version in the directory name. Therefore, we use the "binstubs" feature to
# create the links to the executable. Unfortunately, if we use the
# "tools/ruby/bin" directory as the target location then the "bundle"
# executable will be overridden and stop working. To work around this problem,
# we use two directories for Ruby binaries. The first contains the binaries
# installed using the "gem" command, and the second is a target for the
# "bundle" command.
ruby_tools_bin_dir = File.join(ruby_tools_dir, "bin")
directory ruby_tools_bin_dir
ruby_tools_bin_bundle_dir = File.join(ruby_tools_dir, "bin_bundle")
directory ruby_tools_bin_bundle_dir

# Automatically created directories by tools
ruby_tools_gems_dir = File.join(ruby_tools_dir, "gems")
gobin = File.join(go_tools_dir, "go", "bin")
python_tools_dir = File.join(tools_dir, "python")
pythonpath = File.join(python_tools_dir, "lib")
node_bin_dir = File.join(node_dir, "bin")
protoc_dir = go_tools_dir

# Environment variables
ENV["GEM_HOME"] = ruby_tools_dir
ENV["BUNDLE_PATH"] = ruby_tools_dir
ENV["BUNDLE_BIN"] = ruby_tools_bin_bundle_dir
ENV["GOPATH"] = gopath
ENV["GOBIN"] = gobin
ENV["PATH"] = "#{node_bin_dir}:#{tools_dir}:#{gobin}:#{ENV["PATH"]}"
ENV["PYTHONPATH"] = pythonpath
ENV["VIRTUAL_ENV"] = python_tools_dir

### Detect Chrome
# CHROME_BIN is required for UI unit tests and system tests. If it is
# not provided by a user, try to locate Chrome binary and set
# environment variable to its location.
def detect_chrome_binary()
    if !ENV['CHROME_BIN'].nil? && !ENV['CHROME_BIN'].empty?
        location = which(ENV['CHROME_BIN'])
        if !location.nil?
            return location
        end
    end

    location = which("chromium")
    if !location.nil?
        return location
    end

    location = which("chrome")
    if !location.nil?
        return location
    end

    chrome_locations = []

    if OS == 'linux'
        chrome_locations = ['/usr/bin/chromium-browser', '/snap/bin/chromium', '/usr/bin/chromium']
    elsif OS == 'macos'
        chrome_locations = ["/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome"]
    end

    # For each possible location check if the binary exists.
    chrome_locations.each do |loc|
        if File.exist?(loc)
            # Found Chrome binary.
            return loc
        end
    end

    return nil
end

CHROME = create_manually_installed_file_task(detect_chrome_binary() || "chrome")
ENV["CHROME_BIN"] = CHROME

# System tools
WGET = require_manual_install_on("wget", any_system)
PYTHON3_SYSTEM = require_manual_install_on("python3", any_system)
JAVA = require_manual_install_on("java", any_system)
UNZIP = require_manual_install_on("unzip", any_system)
ENTR = require_manual_install_on("entr", any_system)
GIT = require_manual_install_on("git", any_system)
CREATEDB = require_manual_install_on("createdb", any_system)
PSQL = require_manual_install_on("psql", any_system)
DROPDB = require_manual_install_on("dropdb", any_system)
DROPUSER = require_manual_install_on("dropuser", any_system)
DOCKER = require_manual_install_on("docker", any_system)
OPENSSL = require_manual_install_on("openssl", any_system)
GEM = require_manual_install_on("gem", any_system)
RUBY = require_manual_install_on("ruby", any_system)
TAR = require_manual_install_on("tar", any_system)
SED = require_manual_install_on("sed", any_system)
PERL = require_manual_install_on("perl", any_system)
FOLD = require_manual_install_on("fold", any_system)
SSH = require_manual_install_on("ssh", any_system)
SCP = require_manual_install_on("scp", any_system)
CLOUDSMITH = require_manual_install_on("cloudsmith", any_system)
ETAGS_CTAGS = require_manual_install_on("etags.ctags", any_system)
CLANGPLUSPLUS = require_manual_install_on("clang++", openbsd_system)
FLAMETHROWER = require_manual_install_on("flame", any_system)
DIG = require_manual_install_on("dig", any_system)
PERFDHCP = require_manual_install_on("perfdhcp", any_system)

# Docker plugins
DOCKER_COMPOSE = docker_plugin("docker-compose", "compose")
DOCKER_BUILDX = docker_plugin("docker-buildx", "buildx")

# Toolkits
# We use the executable located in the "gems" directory instead of the one in
# the "bin" directory because the binary in the "bin" directory is the same as
# installed in the system. It isn't an executable installed by the below
# "gem install" command.
BUNDLE = File.join(ruby_tools_gems_dir, "bundler-#{bundler_ver}", "exe", "bundle")
file BUNDLE => [RUBY, GEM, ruby_tools_dir, ruby_tools_bin_dir] do
    sh "rm", "-rf", File.join(ruby_tools_dir, "*")

    sh GEM, "install",
            "--minimal-deps",
            "--no-document",
            "--no-user-install",
            "--install-dir", ruby_tools_dir,
            "bundler:#{bundler_ver}"

    sh "touch", "-c", BUNDLE
    sh BUNDLE, "--version"
end
add_hash_guard(BUNDLE, RUBY)

danger_gemfile = File.expand_path("init_deps/danger/Gemfile", __dir__)
DANGER = File.join(ruby_tools_bin_bundle_dir, "danger")
file DANGER => [ruby_tools_bin_bundle_dir, ruby_tools_dir, BUNDLE] do
    sh BUNDLE, "install",
        "--gemfile", danger_gemfile,
        "--path", ruby_tools_dir,
        "--binstubs", ruby_tools_bin_bundle_dir
    sh "touch", "-c", DANGER
    sh DANGER, "--version"
end
add_hash_guard(DANGER, danger_gemfile)

node = File.join(node_bin_dir, "node")
file node => [TAR, WGET, node_dir] do
    Dir.chdir(node_dir) do
        FileUtils.rm_rf(FileList["*"])
        fetch_file "https://nodejs.org/dist/v#{node_ver}/node-v#{node_ver}-#{node_suffix}.tar.xz", "node.tar.xz"
        sh TAR, "-Jxf", "node.tar.xz", "--strip-components=1"
        sh "rm", "node.tar.xz"
    end
    sh "touch", "-c", node
    sh node, "--version"
end
node = require_manual_install_on(node, libc_musl_system, freebsd_system, openbsd_system)
add_version_guard(node, node_ver)

npm = File.join(node_bin_dir, "npm")
file npm => [node] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    # NPM is initially installed with NodeJS.
    sh npm, "install",
            "-g",
            *ci_opts,
            "npm@#{npm_ver}"
    sh "touch", "-c", npm
    sh npm, "--version"
end
NPM = require_manual_install_on(npm, libc_musl_system, freebsd_system, openbsd_system)
add_version_guard(NPM, npm_ver)

npx = File.join(node_bin_dir, "npx")
file npx => [NPM] do
    sh npx, "--version"
    sh "touch", "-c", npx
end
NPX = require_manual_install_on(npx, libc_musl_system, freebsd_system, openbsd_system)

YAMLINC = File.join(node_dir, "node_modules", "lib", "node_modules", "yamlinc", "bin", "yamlinc")
file YAMLINC => [NPM] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    sh NPM, "install",
            "-g",
            *ci_opts,
            "--prefix", "#{node_dir}/node_modules",
            "yamlinc@#{yamlinc_ver}"
    sh "touch", "-c", YAMLINC
    sh YAMLINC, "--version"
end
add_version_guard(YAMLINC, yamlinc_ver)

STORYBOOK = File.join(node_dir, "node_modules", "bin", "sb")
file STORYBOOK => [NPM] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    sh NPM, "install",
            "-g",
            *ci_opts,
            "--prefix", "#{node_dir}/node_modules",
            "storybook@#{storybook_ver}"
    sh "touch", "-c", STORYBOOK
    sh STORYBOOK, "--version"
end
add_version_guard(STORYBOOK, storybook_ver)

OPENAPI_GENERATOR = File.join(tools_dir, "openapi-generator-cli.jar")
file OPENAPI_GENERATOR => [WGET, tools_dir] do
    fetch_file "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/#{openapi_generator_ver}/openapi-generator-cli-#{openapi_generator_ver}.jar", OPENAPI_GENERATOR
    sh "touch", "-c", OPENAPI_GENERATOR
end
add_version_guard(OPENAPI_GENERATOR, openapi_generator_ver)

go = File.join(gobin, "go")
file go => [WGET, go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        FileUtils.rm_rf("go")
        fetch_file "https://dl.google.com/go/go#{go_ver}.#{go_suffix}.tar.gz", "go.tar.gz"
        sh "tar", "-zxf", "go.tar.gz"
        sh "rm", "go.tar.gz"
    end
    sh "touch", "-c", go
    sh go, "version"
end
GO = require_manual_install_on(go, openbsd_system, freebsd_arm64_system)
add_version_guard(GO, go_ver)

GOSWAGGER = File.join(go_tools_dir, "goswagger")
file GOSWAGGER => [WGET, GO, TAR, go_tools_dir] do
    if OS != 'FreeBSD' && OS != "OpenBSD"
        fetch_file "https://github.com/go-swagger/go-swagger/releases/download/#{goswagger_ver}/swagger_#{goswagger_suffix}", GOSWAGGER
        sh "chmod", "u+x", GOSWAGGER
    else
        # GoSwagger lacks the packages for BSD-like systems then it must be
        # built from sources.
        goswagger_archive = "#{GOSWAGGER}.tar.gz"
        goswagger_dir = "#{GOSWAGGER}-sources"
        sh "mkdir", goswagger_dir
        fetch_file "https://github.com/go-swagger/go-swagger/archive/refs/tags/#{goswagger_ver}.tar.gz", goswagger_archive
        sh TAR, "-zxf", goswagger_archive, "-C", goswagger_dir
        # We cannot use --strip-components because OpenBSD tar doesn't support it.
        goswagger_dir = File.join(goswagger_dir, "go-swagger-#{goswagger_ver[1..-1]}") # Trim 'v' letter
        goswagger_build_dir = File.join(goswagger_dir, "cmd", "swagger")
        Dir.chdir(goswagger_build_dir) do
            sh GO, "build", "-ldflags=-X 'github.com/go-swagger/go-swagger/cmd/swagger/commands.Version=#{goswagger_ver}'"
        end
        sh "mv", File.join(goswagger_build_dir, "swagger"), GOSWAGGER
        sh "rm", "-rf", goswagger_dir
        sh "rm", goswagger_archive
    end

    sh "touch", "-c", GOSWAGGER
    sh GOSWAGGER, "version"
end
add_version_guard(GOSWAGGER, goswagger_ver)

protoc = File.join(protoc_dir, "protoc")
file protoc => [WGET, UNZIP, go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        fetch_file "https://github.com/protocolbuffers/protobuf/releases/download/v#{protoc_ver}/protoc-#{protoc_ver}-#{protoc_suffix}.zip", "protoc.zip"
        sh UNZIP, "-o", "-j", "protoc.zip", "bin/protoc"
        sh "rm", "protoc.zip"
    end
    sh protoc, "--version"
    sh "touch", "-c", protoc
end
PROTOC = require_manual_install_on(protoc, freebsd_system, openbsd_system)
add_version_guard(PROTOC, protoc_ver)

PROTOC_GEN_GO = File.join(gobin, "protoc-gen-go")
file PROTOC_GEN_GO => [GO] do
    sh GO, "install", "google.golang.org/protobuf/cmd/protoc-gen-go@#{protoc_gen_go_ver}"
    sh PROTOC_GEN_GO, "--version"
end
add_version_guard(PROTOC_GEN_GO, protoc_gen_go_ver)

PROTOC_GEN_GO_GRPC = File.join(gobin, "protoc-gen-go-grpc")
file PROTOC_GEN_GO_GRPC => [GO] do
    sh GO, "install", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@#{protoc_gen_go_grpc_ver}"
    sh PROTOC_GEN_GO_GRPC, "--version"
end
add_version_guard(PROTOC_GEN_GO_GRPC, protoc_gen_go_grpc_ver)

golangcilint = File.join(go_tools_dir, "golangci-lint")
file golangcilint => [WGET, GO, TAR, go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        fetch_file "https://github.com/golangci/golangci-lint/releases/download/v#{golangcilint_ver}/golangci-lint-#{golangcilint_ver}-#{golangcilint_suffix}.tar.gz", "golangci-lint.tar.gz"
        sh "mkdir", "tmp"
        sh TAR, "-zxf", "golangci-lint.tar.gz", "-C", "tmp", "--strip-components=1"
        sh "mv", "tmp/golangci-lint", "."
        sh "rm", "-rf", "tmp"
        sh "rm", "-f", "golangci-lint.tar.gz"
    end
    sh "touch", "-c", golangcilint
    sh golangcilint, "--version"
end
GOLANGCILINT = require_manual_install_on(golangcilint, openbsd_system)
add_version_guard(GOLANGCILINT, golangcilint_ver)

GOLIVEPPROF = File.join(gobin, "live-pprof")
file GOLIVEPPROF => [GO] do
    sh GO, "install", "github.com/moderato-app/live-pprof@#{go_live_pprof_ver}"
    if !File.file?(GOLIVEPPROF)
        fail
    end
end
add_version_guard(GOLIVEPPROF, go_live_pprof_ver)

shellcheck = File.join(tools_dir, "shellcheck")
file shellcheck => [WGET, TAR, tools_dir] do
    Dir.chdir(tools_dir) do
        # Download the shellcheck binary.
        fetch_file "https://github.com/koalaman/shellcheck/releases/download/v#{shellcheck_ver}/shellcheck-v#{shellcheck_ver}.#{shellcheck_suffix}.tar.xz", "shellcheck.tar.xz"
        sh "mkdir", "-p", "tmp"
        sh TAR, "-xf", "shellcheck.tar.xz", "-C", "tmp", "--strip-components=1"
        sh "mv", "tmp/shellcheck", "."
        sh "rm", "-rf", "tmp"
        sh "rm", "-f", "shellcheck.tar.xz"
    end
    sh "touch", "-c", shellcheck
    sh shellcheck, "--version"
end
SHELLCHECK = require_manual_install_on(shellcheck, freebsd_system, openbsd_system, macos_arm64_system)
add_version_guard(SHELLCHECK, shellcheck_ver)

TPARSE = "#{gobin}/tparse"
file TPARSE => [GO] do
    sh GO, "install", "github.com/mfridman/tparse@#{tparse_ver}"
    sh TPARSE, "--version"
end
add_version_guard(TPARSE, tparse_ver)

GO_JUNIT_REPORT = "#{gobin}/go-junit-report"
file GO_JUNIT_REPORT => [GO] do
    sh GO, "install", "github.com/jstemmer/go-junit-report/v2@#{go_junit_report_ver}"
    sh GO_JUNIT_REPORT, "--version"
end
add_version_guard(GO_JUNIT_REPORT, go_junit_report_ver)

GOCOVER_COBERTURA = "#{gobin}/gocover-cobertura"
file GOCOVER_COBERTURA => [GO] do
    sh GO, "install", "github.com/boumenot/gocover-cobertura@#{gocover_cobertura_ver}"
    if !File.file?(GOCOVER_COBERTURA)
        fail
    end
end

MOCKGEN = File.join(gobin, "mockgen")
file MOCKGEN => [GO] do
    sh GO, "install", "go.uber.org/mock/mockgen@#{mockgen_ver}"
    sh MOCKGEN, "--version"
end
add_version_guard(MOCKGEN, mockgen_ver)

DLV = File.join(gobin, "dlv")
file DLV => [GO] do
    sh GO, "install", "github.com/go-delve/delve/cmd/dlv@#{dlv_ver}"
    sh DLV, "version"
end
add_version_guard(DLV, dlv_ver)

GDLV = File.join(gobin, "gdlv")
file GDLV => [GO] do
    sh GO, "install", "github.com/aarzilli/gdlv@#{gdlv_ver}"
    if !File.file?(GDLV)
        fail
    end
end
add_version_guard(GDLV, gdlv_ver)

NFPM = File.join(gobin, "nfpm")
file NFPM => [GO] do
    nfpm_major_ver = nfpm_ver.split('.')[0]
    sh GO, "install", "github.com/goreleaser/nfpm/#{nfpm_major_ver}/cmd/nfpm@#{nfpm_ver}"
    sh NFPM, "--version"
end
add_version_guard(NFPM, nfpm_ver)

GOVULNCHECK = File.join(gobin, "govulncheck")
file GOVULNCHECK => [GO] do
    sh GO, "install", "golang.org/x/vuln/cmd/govulncheck@#{govulncheck_ver}"
    sh GOVULNCHECK, "-version"
    sh "touch", "-c", GOVULNCHECK
end
add_version_guard(GOVULNCHECK, govulncheck_ver)

PYTHON = File.join(python_tools_dir, "bin", "python")
file PYTHON => [PYTHON3_SYSTEM] do
    sh "rm", "-rf", File.join(python_tools_dir, "*")
    sh PYTHON3_SYSTEM, "-m", "venv", python_tools_dir
    sh PYTHON, "--version"
end
add_hash_guard(PYTHON, PYTHON3_SYSTEM)

PIP = File.join(python_tools_dir, "bin", "pip")
file PIP => [PYTHON] do
    sh PYTHON, "-m", "ensurepip", "-U", "--default-pip"
    sh "touch", "-c", PIP
    sh PIP, "install", "--prefer-binary", "wheel"
    sh PIP, "--version"
end

SPHINX_BUILD = File.join(python_tools_dir, "bin", "sphinx-build")
sphinx_requirements_file = File.expand_path("init_deps/sphinx.txt", __dir__)
file SPHINX_BUILD => [PIP] do
    sh PIP, "install", "--prefer-binary", "-r", sphinx_requirements_file
    sh "touch", "-c", SPHINX_BUILD
    sh SPHINX_BUILD, "--version"
end
add_hash_guard(SPHINX_BUILD, sphinx_requirements_file)

PYTEST = File.join(python_tools_dir, "bin", "pytest")
pytests_requirements_file = File.expand_path("init_deps/pytest.txt", __dir__)
file PYTEST => [PIP] do
    sh PIP, "install", "--prefer-binary", "-r", pytests_requirements_file
    sh "touch", "-c", PYTEST
    sh PYTEST, "--version"
end
add_hash_guard(PYTEST, pytests_requirements_file)

PROTOC_GEN_PYTHON_GRPC = File.join(python_tools_dir, "bin", "protoc-gen-python_grpc")
file PROTOC_GEN_PYTHON_GRPC => [PYTEST] do
    sh "touch", "-c", PROTOC_GEN_PYTHON_GRPC
    if !File.file?(PROTOC_GEN_PYTHON_GRPC)
        # This plugin doesn't support version printing.
        fail
    end
end

PIP_COMPILE = File.join(python_tools_dir, "bin", "pip-compile")
file PIP_COMPILE => [PIP] do
    sh PIP, "install", "--prefer-binary", "pip-tools==#{pip_tools_ver}"
    sh "touch", "-c", PIP_COMPILE
    sh PIP_COMPILE, "--version"
end
add_version_guard(PIP_COMPILE, pip_tools_ver)

PIP_SYNC = File.join(python_tools_dir, "bin", "pip-sync")
file PIP_SYNC => [PIP_COMPILE]

PIP_AUDIT = File.join(python_tools_dir, "bin", "pip-audit")
file PIP_AUDIT => [PIP] do
    sh PIP, "install", "pip-audit==#{pip_audit_ver}"
    sh "touch", "-c", PIP_AUDIT
    sh PIP_AUDIT, "--version"
end

PYLINT = File.join(python_tools_dir, "bin", "pylint")
python_linters_requirements_file = File.expand_path("init_deps/pylinters.txt", __dir__)
file PYLINT => [PIP] do
    sh PIP, "install", "--prefer-binary", "-r", python_linters_requirements_file
    sh "touch", "-c", PYLINT
    sh PYLINT, "--version"
end
add_hash_guard(PYLINT, python_linters_requirements_file)

FLAKE8 = File.join(python_tools_dir, "bin", "flake8")
file FLAKE8 => [PYLINT] do
    sh "touch", "-c", FLAKE8
    sh FLAKE8, "--version"
end

BLACK = File.join(python_tools_dir, "bin", "black")
file BLACK => [PYLINT] do
    sh "touch", "-c", BLACK
    sh BLACK, "--version"
end

flask_requirements_file = File.expand_path("init_deps/flask.txt", __dir__)
FLASK = File.join(python_tools_dir, "bin", "flask")
file FLASK => [PIP] do
    sh PIP, "install", "--prefer-binary", "-r", flask_requirements_file
    sh "touch", "-c", FLASK
    sh FLASK, "--version"
end
add_hash_guard(FLASK, flask_requirements_file)

#############
### Tasks ###
#############

desc 'Install all system-level dependencies'
task :prepare do
    find_and_prepare_deps(__FILE__)
end

desc 'Check all system-level dependencies'
task :check do
    check_deps(__FILE__)
end
