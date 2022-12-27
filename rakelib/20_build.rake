# coding: utf-8

# Build 
# This file is responsible for building (compiling)
# the binaries and other artifacts (docs, bundles). 

############
### Date ###
############

require 'date'

now = Time.now
build_date = now.strftime("%Y-%m-%d %H:%M")

if ENV['STORK_BUILD_TIMESTAMP']
    TIMESTAMP = ENV['STORK_BUILD_TIMESTAMP']
else
    TIMESTAMP = now.strftime("%y%m%d%H%M%S")
end

#####################
### Documentation ###
#####################

ARM_DIRECTORY = "doc/_build/html"
file ARM_DIRECTORY => DOC_CODEBASE + [SPHINX_BUILD] do
    sh SPHINX_BUILD, "-M", "html", "doc/", "doc/_build", "-v", "-E", "-a", "-W", "-j", "2"
    sh "touch", "-c", ARM_DIRECTORY
end
CLEAN.append ARM_DIRECTORY

TOOL_MAN_FILE = "doc/man/stork-tool.8"
file TOOL_MAN_FILE => DOC_CODEBASE + [SPHINX_BUILD] do
    sh SPHINX_BUILD, "-M", "man", "doc/", "doc/", "-v", "-E", "-a", "-W", "-j", "2"
    sh "touch", "-c", TOOL_MAN_FILE, AGENT_MAN_FILE, SERVER_MAN_FILE
end

AGENT_MAN_FILE = "doc/man/stork-agent.8"
file AGENT_MAN_FILE => [TOOL_MAN_FILE]

SERVER_MAN_FILE = "doc/man/stork-server.8"
file SERVER_MAN_FILE => [TOOL_MAN_FILE]

man_files = FileList[SERVER_MAN_FILE, AGENT_MAN_FILE, TOOL_MAN_FILE]
CLEAN.append *man_files

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
file WEBUI_DIST_ARM_DIRECTORY => [ARM_DIRECTORY] do
    sh "cp", "-a", ARM_DIRECTORY, WEBUI_DIST_ARM_DIRECTORY
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
        sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{build_date}'"
    end
    puts "Stork Agent build date: #{build_date} (timestamp: #{TIMESTAMP})"
end
CLEAN.append AGENT_BINARY_FILE

SERVER_BINARY_FILE = "backend/cmd/stork-server/stork-server"
file SERVER_BINARY_FILE => GO_SERVER_CODEBASE + [GO] do
    sh "rm", "-f", GO_SERVER_API_MOCK
    Dir.chdir("backend/cmd/stork-server") do
        sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{build_date}'"
    end
    puts "Stork Server build date: #{build_date} (timestamp: #{TIMESTAMP})"
end
CLEAN.append SERVER_BINARY_FILE

TOOL_BINARY_FILE = "backend/cmd/stork-tool/stork-tool"
file TOOL_BINARY_FILE => GO_TOOL_CODEBASE + [GO] do
    Dir.chdir("backend/cmd/stork-tool") do
        sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{build_date}'"
    end
    puts "Stork Tool build date: #{build_date} (timestamp: #{TIMESTAMP})"
end
CLEAN.append TOOL_BINARY_FILE

CODE_GEN_BINARY_FILE = "backend/cmd/stork-code-gen/stork-code-gen"
file CODE_GEN_BINARY_FILE => GO_CODE_GEN_CODEBASE + [GO] do
    Dir.chdir("backend/cmd/stork-code-gen") do
        sh GO, "build", "-ldflags=-X 'isc.org/stork.BuildDate=#{build_date}'"
    end
    puts "Stork Code Gen build date: #{build_date} (timestamp: #{TIMESTAMP})"
end
CLEAN.append CODE_GEN_BINARY_FILE

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
    desc "Build Stork documentation from sources"
    task :doc => man_files + [ARM_DIRECTORY]

    desc "Build Stork Server from sources"
    task :server => [SERVER_BINARY_FILE]

    desc "Build Stork Agent from sources"
    task :agent => [AGENT_BINARY_FILE]

    desc "Build Stork Tool from sources"
    task :tool => [TOOL_BINARY_FILE]

    desc "Build Stork Code Gen from sources"
    task :code_gen => [CODE_GEN_BINARY_FILE]

    desc "Build Web UI (production mode)"
    task :ui => [WEBUI_DIST_DIRECTORY, WEBUI_DIST_ARM_DIRECTORY]

    desc "Build Stork Backend (Code Generator, Server, Agent, Tool)"
    task :backend => [:code_gen, :server, :agent, :tool]
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
    PORT - agent port to use - default: 8888"
    task :agent => [AGENT_BINARY_FILE] do
        if ENV["PORT"].nil?
            ENV["PORT"] = "8888"
        end
        sh AGENT_BINARY_FILE, "--port", ENV["PORT"]
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
