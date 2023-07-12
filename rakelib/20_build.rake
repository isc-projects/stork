# coding: utf-8

# Build
# This file is responsible for building (compiling)
# the binaries and other artifacts (docs, bundles).

# Defines the operating system and architecture combination guard for a file
# task. This guard allows file tasks to depend on the os and architecture used
# to build the Go binaries.
# The operating system is specified by the STORK_GOOS environment variable. If
# it is not set, the current OS type is used.
# The architecture is specified by the STORK_GOARCH environment variable. If
# it is not set, the current architecture is used.
# The function accepts a task to be guarded.
def add_go_os_arch_guard(task_name)
    arch = ENV["STORK_GOARCH"] || ARCH
    
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
# and STORK_GOARCH environment variables or the current operating system and
# architecture.
def with_custom_go_os_and_arch(&block)
    ENV["GOOS"] = ENV["STORK_GOOS"]
    ENV["GOARCH"] = ENV["STORK_GOARCH"]

    yield

    ENV["GOOS"] = nil
    ENV["GOARCH"] = nil
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

AGENT_BINARY_FILE = "backend/cmd/stork-agent/stork-agent"
file AGENT_BINARY_FILE => GO_AGENT_CODEBASE + [GO] do
    Dir.chdir("backend/cmd/stork-agent") do
        with_custom_go_os_and_arch do
            sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{CURRENT_DATE}'"
        end
    end
    sh "touch", "-c", AGENT_BINARY_FILE
    puts "Stork Agent build date: #{CURRENT_DATE} (timestamp: #{TIMESTAMP})"
end
add_go_os_arch_guard(AGENT_BINARY_FILE)
CLEAN.append AGENT_BINARY_FILE

SERVER_BINARY_FILE = "backend/cmd/stork-server/stork-server"
file SERVER_BINARY_FILE => GO_SERVER_CODEBASE + [GO] do
    Dir.chdir("backend/cmd/stork-server") do
        with_custom_go_os_and_arch do
            sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{CURRENT_DATE}'"
        end
    end
    sh "touch", "-c", SERVER_BINARY_FILE
    puts "Stork Server build date: #{CURRENT_DATE} (timestamp: #{TIMESTAMP})"
end
add_go_os_arch_guard(SERVER_BINARY_FILE)
CLEAN.append SERVER_BINARY_FILE

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
            if File.exists? WEBUI_DIST_DIRECTORY
                production_time = File.mtime(WEBUI_DIST_DIRECTORY)
            end
            testing_time = Time.new(1980, 1, 1)
            if File.exists? WEBUI_DEBUG_DIRECTORY
                testing_time = File.mtime(WEBUI_DEBUG_DIRECTORY)
            end
            use_testing_ui = testing_time > production_time
            puts "Using testing UI - testing UI is newer than production"
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
        sh "touch", "-c", "backend/cmd/stork-server"
        Rake::Task["build:server"].invoke()
    end

    desc "Rebuild Stork Agent from sources"
    task :agent do
        sh "touch", "-c", "backend/cmd/stork-agent"
        Rake::Task["build:agent"].invoke()
    end

    desc "Rebuild Stork Tool from sources"
    task :tool do
        sh "touch", "-c", "backend/cmd/stork-tool"
        Rake::Task["build:tool"].invoke()
    end

    desc "Rebuild Web UI (production mode)"
    task :ui do
        sh "touch", "-c", "webui"
        Rake::Task["build:ui"].invoke()
    end
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
        end

        sh AGENT_BINARY_FILE, *opts
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
