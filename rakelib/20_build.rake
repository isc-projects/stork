# coding: utf-8

# Build
# This file is responsible for building (compiling)
# the binaries and other artifacts (docs, bundles).

# Returns the conventional architecture name based on the target architecture
# of the Golang binaries. The architecture is specified by the STORK_GOARCH and
# (optionally) STORK_GOARM environment variables. If they are not set, the
# current architecture is used.
#
# Note that if the 32-bit ARM version is provided in the STORK_GOARM variable,
# the architecture from the STORK_GOARCH or the current architecture is not
# validated to be 32-bit.
def get_target_go_arch()
    arch = ENV["STORK_GOARCH"] || ARCH
    arm_version_raw = ENV["STORK_GOARM"]
    if !arm_version_raw.nil?
        arm_version = arm_version_raw.to_i
        # The above architecture suffixes were not tested on BSD systems.
        # They may not be suitable for this operating system family.
        case arm_version
        when 0
            fail "STORK_GOARM must be a number, got: #{arm_version_raw}"
        when 5
            arch = "armel"
        when 6..7
            arch = "armhf"
        when 8
            puts "STORK_GOARM is ignored for 64-bit ARM (armv8)"
        else
            fail "Unsupported STORK_GOARM value: #{arm_version_raw}"
        end
    end
    arch
end

# Defines the operating system and architecture combination guard for a file
# task. This guard allows file tasks to depend on the os and architecture used
# to build the Go binaries.
# The operating system is specified by the STORK_GOOS environment variable. If
# it is not set, the current OS type is used.
# The architecture is specified by the STORK_GOARCH environment variable. If
# it is not set, the current architecture is used.
# The ARM architecture is specified by the STORK_GOARM environment variable.
# If it is not set, it is not used. It does not affect to `arm64`.
# The function accepts a task to be guarded.
def add_go_os_arch_guard(task_name)
    arch = get_target_go_arch()

    os = ENV["STORK_GOOS"]
    if os.nil?
        case OS
        when "macos"
            os = "darwin"
        when "linux"
            os = "linux"
        when "FreeBSD"
            os = "freebsd"
        when "OpenBSD"
            os = "openbsd"
        else
            puts "ERROR: Operating system is not supported: #{OS}"
            fail
        end
    end

    identifier = "#{os}-#{arch}"
    add_guard(task_name, identifier, "os-arch")
end

# Runs a given block with the GOOS and GOARCH environment variables for the
# Golang compiler. The values of the variables are set based on the STORK_GOOS
# and STORK_GOARCH (and optionally STORK_GOARM) environment variables or the
# current operating system and architecture.
def with_custom_go_os_and_arch(&block)
    ENV["GOOS"] = ENV["STORK_GOOS"]
    ENV["GOARCH"] = ENV["STORK_GOARCH"]
    ENV["GOARM"] = ENV["STORK_GOARM"]

    yield

    ENV["GOOS"] = nil
    ENV["GOARCH"] = nil
    ENV["GOARM"] = nil
end

# The prerequisites of the given task can be suppressed by setting the
# SUPPRESS_PREREQUISITES environment variable to "true". It should be helpful
# to run tasks that direct prerequisites exist (for example, they were built
# in another environment), but the nested ones do not.
def allow_suppress_prerequisites(task_name)
    if ENV["SUPPRESS_PREREQUISITES"] == "true"
        puts "Suppressing prerequisites for #{task_name}"
        Rake::Task[task_name].clear_prerequisites()
    end
end

#####################
### Documentation ###
#####################

DOC_USER_ROOT = "doc/build/user/html"
file DOC_USER_ROOT => [SPHINX_BUILD] + DOC_USER_CODEBASE do
    sh SPHINX_BUILD,
        "-M", "html",
        "doc/user", "doc/build/user",
        "-v",
        "-E",
        "-a",
        "-W",
        "-j", "2"
    sh "touch", "-c", DOC_USER_ROOT
end

TOOL_MAN_FILE = "doc/build/man/stork-tool.8"
file TOOL_MAN_FILE => [SPHINX_BUILD] + DOC_USER_CODEBASE do
    sh SPHINX_BUILD,
        "-M", "man",
        "doc/user", "doc/build",
        "-v",
        "-E",
        "-a",
        "-W",
        "-j", "2"
    sh "touch", "-c", TOOL_MAN_FILE, AGENT_MAN_FILE, SERVER_MAN_FILE
end

AGENT_MAN_FILE = "doc/build/man/stork-agent.8"
file AGENT_MAN_FILE => [TOOL_MAN_FILE]

SERVER_MAN_FILE = "doc/build/man/stork-server.8"
file SERVER_MAN_FILE => [TOOL_MAN_FILE]

DOC_DEV_ROOT = "doc/build/dev/html"
file DOC_DEV_ROOT => DOC_DEV_CODEBASE + [SPHINX_BUILD] do
    sh SPHINX_BUILD,
        "-M", "html",
        "doc/dev", "doc/build/dev",
        "-v",
        "-E",
        "-a",
        "-W",
        "-j", "2"
    sh "touch", "-c", DOC_DEV_ROOT
end

CLEAN.append "doc/build"

################
### Frontend ###
################

file WEBUI_DIST_DIRECTORY = "webui/dist/stork"
file WEBUI_DIST_DIRECTORY => WEBUI_CODEBASE + [NPX] do
    Dir.chdir("webui") do
        sh NPX, "ng", "build", "--configuration", "production"
    end
end

file WEBUI_DIST_ARM_DIRECTORY = "webui/dist/stork/assets/arm"
file WEBUI_DIST_ARM_DIRECTORY => [DOC_USER_ROOT] do
    sh "cp", "-a", DOC_USER_ROOT, WEBUI_DIST_ARM_DIRECTORY
    sh "touch", "-c", WEBUI_DIST_ARM_DIRECTORY
end

file WEBUI_DEBUG_DIRECTORY = "webui/dist/stork-debug"
file WEBUI_DEBUG_DIRECTORY => WEBUI_CODEBASE + [NPX] do
    Dir.chdir("webui") do
        sh NPX, "ng", "build"
    end
end

CLEAN.append "webui/dist"
CLEAN.append "webui/.angular"

###############
### Backend ###
###############

# This helper is used to create the Stork agent file tasks. It allows to specify
# build tags for the conditional compilation. The tags are specified in the
# filename after the plus sign. For example, the filename "stork-agent+profiler"
# will be compiled with the "profiler" build tag.
#
# This helper logic was initially implemented as a Rake rule. However, we have
# rejected this approach because of the internal details of the Rake rule. When the
# rule constructs the task, it checks if the file prerequisites exist and fails
# if they do not. It is done immediately when the given target is accessed by
# `Rake::Task[name]`, not when the task is executed or needed.
# It is problematic when the rule-based task is enhanced by `add_guard` helper.
# This helper works on the task object, so the rule is executed at the same
# time as Rake lists the tasks. It implies that all Stork agent file prerequisites
# must always exist, even if the task not related to agent is executed.
# It is problematic in Docker builds where certain images exclude the agent
# files (e.g., UI builder image).
def stork_agent_conditional(*tags)
    filename = "stork-agent"
    tags.each do |tag|
        filename += "+#{tag}"
    end
    task_name = File.join("backend/cmd/stork-agent", filename)

    file task_name => [GO] + GO_AGENT_CODEBASE do |t|
        Dir.chdir("backend/cmd/stork-agent") do
            with_custom_go_os_and_arch do
                sh GO, "build",
                    "-ldflags=-X 'isc.org/stork.BuildDate=#{CURRENT_DATE}'",
                    "-tags", tags.join(","),
                    "-o", filename
            end
        end
        sh "touch", "-c", t.name
        puts "Stork Agent build date: #{CURRENT_DATE} (timestamp: #{TIMESTAMP})"
    end

    add_go_os_arch_guard(task_name)
    allow_suppress_prerequisites(task_name)
    CLEAN.append task_name

    task_name
end

# The standard Stork agent file task. It is compiled without any custom build
# tags. It is dedicated to release builds.
AGENT_BINARY_FILE = stork_agent_conditional()

# The Stork agent file task compiled with the profiler that allows to profile
# the agent on demand.
AGENT_BINARY_FILE_WITH_PROFILER = stork_agent_conditional("profiler")

# This rule is used to create the Stork server file tasks. It allows to specify
# build tags for the conditional compilation. The tags are specified in the
# filename after the plus sign. For example, the filename "stork-server+profiler"
# will be compiled with the "profiler" build tag.
#
# See `stork_agent_conditional` description for the explanation why this helper
# was not implemented as a rule.
def stork_server_conditional(*tags)
    filename = "stork-server"
    tags.each do |tag|
        filename += "+#{tag}"
    end
    task_name = File.join("backend/cmd/stork-server", filename)

    file task_name => [GO] + GO_SERVER_CODEBASE do |t|
        Dir.chdir("backend/cmd/stork-server") do
            with_custom_go_os_and_arch do
                sh GO, "build",
                    "-ldflags=-X 'isc.org/stork.BuildDate=#{CURRENT_DATE}'",
                    "-tags", tags.join(","),
                    "-o", filename
            end
        end
        sh "touch", "-c", t.name
        puts "Stork Server build date: #{CURRENT_DATE} (timestamp: #{TIMESTAMP})"
    end

    add_go_os_arch_guard(task_name)
    allow_suppress_prerequisites(task_name)
    CLEAN.append task_name

    task_name
end

# The standard Stork server file task. It is compiled without any custom build
# tags. It is dedicated to release builds.
SERVER_BINARY_FILE = stork_server_conditional()

# The Stork server file task compiled with the profiler that allows to profile
# the agent on demand.
SERVER_BINARY_FILE_WITH_PROFILER = stork_server_conditional("profiler")

TOOL_BINARY_FILE = "backend/cmd/stork-tool/stork-tool"
file TOOL_BINARY_FILE => GO_TOOL_CODEBASE + [GO] do
    Dir.chdir("backend/cmd/stork-tool") do
        with_custom_go_os_and_arch do
            sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{CURRENT_DATE}'"
        end
    end
    sh "touch", "-c", TOOL_BINARY_FILE
    puts "Stork Tool build date: #{CURRENT_DATE} (timestamp: #{TIMESTAMP})"
end
add_go_os_arch_guard(TOOL_BINARY_FILE)
allow_suppress_prerequisites(TOOL_BINARY_FILE)
CLEAN.append TOOL_BINARY_FILE

#############
### Tasks ###
#############

# Internal task that configures environment variables for server
task :pre_run_server do
    if ENV["DB_TRACE"] == "true"
        ENV["STORK_DATABASE_TRACE"] = "run"
    end

    ui_mode = ENV["UI_MODE"]

    use_testing_ui = false
    # If the UI mode is not provided then detect it
    if ui_mode == nil
        # Enable testing mode if live build UI is active
        use_testing_ui = system "pgrep", "-f", "ng build --watch"
        # Enable testing mode if testing dir is newer then production dir
        if use_testing_ui == true
            puts "Using testing UI - live UI build is active"
        else
            production_time = Time.new(1980, 1, 1)
            if File.exist? WEBUI_DIST_DIRECTORY
                production_time = File.mtime(WEBUI_DIST_DIRECTORY)
            end
            testing_time = Time.new(1980, 1, 1)
            if File.exist? WEBUI_DEBUG_DIRECTORY
                testing_time = File.mtime(WEBUI_DEBUG_DIRECTORY)
            end
            use_testing_ui = testing_time > production_time
            if use_testing_ui
                puts "Using testing UI - testing UI is newer than production"
            else
                puts "Using production UI - production UI is newer than testing"
            end
        end
    elsif ui_mode == "testing"
        # Check if user manually forces the UI mode
        use_testing_ui = true
        puts "Using testing UI - user choice"
    elsif ui_mode == "none"
        puts "Skip UI - user choice"
    elsif ui_mode != "production"
        puts "Invalid UI mode - choose 'production', 'testing' or unspecify"
        fail
    end

    # Set environment variables
    if use_testing_ui
        ENV["STORK_REST_STATIC_FILES_DIR"] = WEBUI_DEBUG_DIRECTORY
    else
        ENV["STORK_REST_STATIC_FILES_DIR"] = WEBUI_DIST_DIRECTORY
    end

    ENV["STORK_SERVER_ENABLE_METRICS"] = "true"

    # Build UI
    if ui_mode != "none"
        Rake::Task[ENV["STORK_REST_STATIC_FILES_DIR"]].invoke()
    end
end

## Build

namespace :build do
    desc "Build user and developer Stork documentation from sources"
    task :doc => ["build:doc:user", "build:doc:dev", "build:doc:man"]

    namespace :doc do
        desc "Build Stork documentation from sources"
        task :user => [DOC_USER_ROOT]

        desc "Build Stork Developer's guide from sources"
        task :dev => [DOC_DEV_ROOT]

        desc "Build Stork man documentation from sources"
        task :man => [TOOL_MAN_FILE, AGENT_MAN_FILE, SERVER_MAN_FILE]
    end

    desc "Build Stork Server from sources"
    task :server => [SERVER_BINARY_FILE]

    desc "Build Stork Agent from sources"
    task :agent => [AGENT_BINARY_FILE]

    desc "Build Stork Tool from sources"
    task :tool => [TOOL_BINARY_FILE]

    desc "Build Web UI (production mode)"
    task :ui => [WEBUI_DIST_DIRECTORY, WEBUI_DIST_ARM_DIRECTORY]

    desc "Build Stork Backend (Server, Agent, Tool)"
    task :backend => [:server, :agent, :tool]

    desc "Build Stork Code Gen from sources"
    task :code_gen => [CODE_GEN_BINARY_FILE]
end

desc "Build all Stork components (Server, Agent, Tool, UI, doc)"
task :build => ["build:backend", "build:doc", "build:ui"]


## Rebuild
namespace :rebuild do
    desc "Rebuild Stork documentation from sources"
    task :doc do
        sh "touch", "-c", "doc"
        Rake::Task["build:doc"].invoke()
    end

    desc "Rebuild Stork Server from sources"
    task :server do
        sh "rm", "-f", SERVER_BINARY_FILE
        Rake::Task["build:server"].invoke()
    end

    desc "Rebuild Stork Agent from sources"
    task :agent do
        sh "rm", "-f", AGENT_BINARY_FILE
        Rake::Task["build:agent"].invoke()
    end

    desc "Rebuild Stork Tool from sources"
    task :tool do
        sh "rm", "-f", TOOL_BINARY_FILE
        Rake::Task["build:tool"].invoke()
    end

    desc "Rebuild Web UI (production mode)"
    task :ui do
        sh "touch", "-c", "webui"
        Rake::Task["build:ui"].invoke()
    end

    desc "Rebuild backend"
    task :backend => ["rebuild:server", "rebuild:agent", "rebuild:tool"]
end

## Run
namespace :run do
    desc "Run Stork Server (release mode)
        UI_MODE - WebUI mode to use - choose: 'production', 'testing', 'none' or unspecify
        DB_TRACE - trace SQL queries - default: false
    "
    task :server => [SERVER_BINARY_FILE, :pre_run_server] do
        sh SERVER_BINARY_FILE
    end

    desc "Run Stork Server (profiler mode)
        UI_MODE - WebUI mode to use - choose: 'production', 'testing', 'none' or unspecify
        DB_TRACE - trace SQL queries - default: false
    "
    task :server_profiling => [SERVER_BINARY_FILE_WITH_PROFILER, :pre_run_server] do
        sh SERVER_BINARY_FILE_WITH_PROFILER
    end

    desc "Run Stork Agent (release mode)
        PORT - agent port to use - default: 8888
        REGISTER - register in the localhost server - default: false"
    task :agent => [AGENT_BINARY_FILE] do
        if ENV["PORT"].nil?
            ENV["PORT"] = "8888"
        end

        opts = ["--port", ENV["PORT"]]

        if ENV["REGISTER"] == "true"
            opts.append "--host", "localhost"
            opts.append "--server-url", "http://localhost:8080"
        else
            opts.append "--listen-prometheus-only"
        end

        sh AGENT_BINARY_FILE, *opts
    end

    desc "Run Stork Agent (profiler mode)
        PORT - agent port to use - default: 8888
        REGISTER - register in the localhost server - default: false"
    task :agent_profiling => [AGENT_BINARY_FILE_WITH_PROFILER] do
        if ENV["PORT"].nil?
            ENV["PORT"] = "8888"
        end

        opts = ["--port", ENV["PORT"]]

        if ENV["REGISTER"] == "true"
            opts.append "--host", "localhost"
            opts.append "--server-url", "http://localhost:8080"
        else
            opts.append "--listen-prometheus-only"
        end

        sh AGENT_BINARY_FILE_WITH_PROFILER, *opts
    end
end

namespace :prepare do
    desc 'Install the external dependencies related to the build'
    task :build do
        find_and_prepare_deps(__FILE__)
    end
end

namespace :check do
    desc 'Check the external dependencies related to the build'
    task :build do
        check_deps(__FILE__)
    end
end
