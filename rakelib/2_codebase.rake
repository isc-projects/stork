# coding: utf-8

###############
### Swagger ###
###############

swagger_file = 'api/swagger.yaml'
swagger_api_files = FileList['api/*.yaml'].exclude(swagger_file)
file swagger_file => swagger_api_files + [YAMLINC] do
    sh YAMLINC, "-o", swagger_file, "api/swagger.in.yaml"
end

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

agent_proto_file = "backend/api/agent.proto"
agent_pb_go_file = "backend/api/agent.pb.go"
agent_grpc_pb_go_file = "backend/api/agent_grpc.pb.go"
file agent_pb_go_file => [agent_proto_file, PROTOC, PROTOC_GEN_GO, PROTOC_GEN_GO_GRPC] do
    Dir.chdir("backend/api") do
        sh PROTOC, "--proto_path=.", "--go_out=.", "--go-grpc_out=.", "agent.proto"
    end
end
file agent_grpc_pb_go_file => [agent_pb_go_file]

go_dependencies_dir = File.join(ENV["GOPATH"], "pkg")
file go_dependencies_dir => [GO, "backend/go.mod", "backend/go.sum"] do
    Dir.chdir("backend") do
        sh GO, "mod", "download"
    end
    sh "touch", go_dependencies_dir
end

go_server_codebase = FileList[
    "backend/server",
    "backend/server/**/*",
    "backend/cmd/stork-server",
    "backend/cmd/stork-server/*",
    swagger_server_dir
]


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
    .exclude(go_server_codebase)
    .exclude(go_agent_codebase)
    .exclude(go_tool_codebase)
    .include(go_dependencies_dir)
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

node_module_dir = "webui/node_modules"
file node_module_dir => [NPM, "webui/package.json", "webui/package-lock.json"] do
    Dir.chdir("webui") do
        ENV["NG_CLI_ANALYTICS"] = "false"
        sh NPM, "ci",
                "--prefer-offline",
                "--no-audit",
                "--no-progress"
    end
    sh "touch", node_module_dir
end

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
