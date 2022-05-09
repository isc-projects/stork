# coding: utf-8

# Codebase
# This file contains definitions of the source files
# including generated ones. It defines convenient file
# lists to use as pre-requirements in the next stages.
# It installs the source code dependencies too.

# Ruby has a built-in solution for handling CLEAN and CLOBBER arrays and
# deleting unnecessary files. But loading the 'rake' module significantly reduces
# the performance. For these reason we implement the clean and clobber tasks
# ourselves.
#
# Clean up the project by deleting scratch files and backup files. Add files to
# the CLEAN FileList to have the clean target handle them.
# Unlike the standard Rake Clean task, this implementation recursively removes
# the directories.
CLEAN = FileList[]
# Clobber all generated and non-source files in a project. The task depends on
# clean, so all the CLEAN files will be deleted as well as files in the CLOBBER
# FileList. The intent of this task is to return a project to its pristine,
# just unpacked state.
CLOBBER = FileList[]

###############
### Swagger ###
###############

swagger_file = 'api/swagger.yaml'
swagger_api_files = FileList['api/*.yaml'].exclude(swagger_file)
file swagger_file => swagger_api_files + [YAMLINC] do
    sh YAMLINC, "-o", swagger_file, "api/swagger.in.yaml"
end
CLEAN.append swagger_file

###############
### Backend ###
###############

swagger_server_dir = "backend/server/gen"
file swagger_server_dir => [swagger_file, GOSWAGGER] do
    swagger_abs = File.expand_path(swagger_file)
    Dir.chdir("backend") do
        sh GOSWAGGER, "generate", "server",
        "-m", "server/gen/models",
        "-s", "server/gen/restapi",
        "--exclude-main",
        "--name", "Stork",
        "--regenerate-configureapi",
        "--spec", swagger_abs,
        "--template", "stratoscale"
    end
    sh "touch", swagger_server_dir
end
CLEAN.append swagger_server_dir

agent_proto_file = "backend/api/agent.proto"
agent_pb_go_file = "backend/api/agent.pb.go"
agent_grpc_pb_go_file = "backend/api/agent_grpc.pb.go"
file agent_pb_go_file => [agent_proto_file, PROTOC, PROTOC_GEN_GO, PROTOC_GEN_GO_GRPC] do
    Dir.chdir("backend/api") do
        sh PROTOC, "--proto_path=.", "--go_out=.", "--go-grpc_out=.", "agent.proto"
    end
end
file agent_grpc_pb_go_file => [agent_pb_go_file]
CLEAN.append agent_pb_go_file, agent_grpc_pb_go_file

# Go dependencies are installed automatically during build
# or can be triggered manually.
CLOBBER.append File.join(ENV["GOPATH"], "pkg")

go_server_codebase = FileList[
    "backend/server",
    "backend/server/**/*",
    "backend/cmd/stork-server",
    "backend/cmd/stork-server/*",
    swagger_server_dir
]
.exclude(swagger_server_dir + "/**/*")

go_agent_codebase = FileList[
    "backend/agent",
    "backend/agent/**/*",
    "backend/cmd/stork-agent",
    "backend/cmd/stork-agent/*",
    "backend/server/certs/**/*",
    "backend/server/database/**/*"
]

go_tool_codebase = FileList[
    "backend/cmd/stork-tool",
    "backend/cmd/stork-tool/*"
]

go_common_codebase = FileList["backend/**/*"]
    .exclude("backend/coverage.out")
    .exclude(swagger_server_dir + "/**/*")
    .exclude(go_server_codebase)
    .exclude(go_agent_codebase)
    .exclude(go_tool_codebase)
    .include(agent_pb_go_file)
    .include(agent_grpc_pb_go_file)

GO_SERVER_API_MOCK = "backend/server/agentcomm/api_mock.go"

GO_SERVER_CODEBASE = go_server_codebase
        .include(go_common_codebase)
        .exclude("backend/cmd/stork-server/stork-server")
        .exclude(GO_SERVER_API_MOCK)

GO_AGENT_CODEBASE = go_agent_codebase
        .include(go_common_codebase)
        .exclude("backend/cmd/stork-agent/stork-agent")

GO_TOOL_CODEBASE = go_tool_codebase
        .include(go_common_codebase)
        .exclude("backend/cmd/stork-tool/stork-tool")

file GO_SERVER_API_MOCK => [GO, MOCKERY, MOCKGEN] + GO_SERVER_CODEBASE do
    Dir.chdir("backend") do
        sh GO, "generate", "-v", "./..."
    end
    sh "touch", GO_SERVER_API_MOCK
end
CLEAN.append GO_SERVER_API_MOCK
    
#####################
### Documentation ###
#####################

DOC_CODEBASE = FileList["doc", "doc/**/*"]
        .include("backend/version.go")
        .exclude("doc/_build")
        .exclude("doc/_build/**/*")
        .exclude("doc/doctrees/**/*")
        .exclude("doc/man/*.8")

################
### Frontend ###
################

open_api_generator_webui_dir = "webui/src/app/backend"
file open_api_generator_webui_dir => [swagger_file, OPENAPI_GENERATOR] do
    sh "java", "-jar", OPENAPI_GENERATOR, "generate",
    "-i", swagger_file,
    "-g", "typescript-angular",
    "-o", open_api_generator_webui_dir,
    "--additional-properties", "snapshot=true,ngVersion=10.1.5,modelPropertyNaming=camelCase"
    sh "touch", open_api_generator_webui_dir
end
CLEAN.append open_api_generator_webui_dir

node_module_dir = "webui/node_modules"
file node_module_dir => [NPM, "webui/package.json", "webui/package-lock.json"] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    Dir.chdir("webui") do
        ENV["NG_CLI_ANALYTICS"] = "false"
        sh NPM, "ci",
                "--prefer-offline",
                *ci_opts
    end
    sh "touch", node_module_dir
end
CLOBBER.append node_module_dir

WEBUI_CODEBASE = FileList["webui", "webui/**/*"]
    .exclude("webui/.angular")
    .exclude("webui/.angular/**/*")
    .exclude("webui/node_modules/**/*")
    .exclude(File.join(open_api_generator_webui_dir, "**/*"))
    .exclude("webui/dist")
    .exclude("webui/dist/**/*")
    .exclude("webui/src/assets/arm")
    .exclude("webui/src/assets/arm/**/*")
    .include(open_api_generator_webui_dir)
    .include(node_module_dir)

#############
### Tasks ###
#############

def remove_files(list)
    list.each do |item|
        FileUtils.rm_rf(item)
    end
end 

namespace :clean do
    desc 'Clean up the project by deleting scratch files and backup files'
    task :soft do
        remove_files(CLEAN)
    end

    desc 'Clobber all generated and non-source files in a project.'
    task :hard => [:soft] do
        remove_files(CLOBBER)
    end
end

namespace :prepare do
    desc 'Install the external dependencies related to the codebase'
    task :codebase do
        find_and_prepare_deps(__FILE__)
    end
    
    desc 'Trigger the backend (GO) dependencies installation.'
    task :backend_deps do
        Dir.chdir("backend") do
            sh GO, "mod", "download"
        end
    end
    
    desc 'Trigger the frontend (UI) dependencies installation'
    task :ui_deps => [node_module_dir]
    
    desc 'Trigger the frontend (UI) and backend (GO) dependencies installation'
    task :deps => [:ui_deps, :backend_deps]
end

namespace :check do
    desc 'Check the external dependencies related to the codebase'
    task :codebase do
        check_deps(__FILE__, "wget", "python3", "pip3", "java", "unzip")
    end
end