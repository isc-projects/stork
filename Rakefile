# coding: utf-8
require 'rake'

# Tool Versions
NODE_VER = '10.16.3'
SWAGGER_CODEGEN_VER = '2.4.8'
GOSWAGGER_VER = 'v0.20.1'
GOLANGCILINT_VER = '1.21.0'
GO_VER = '1.13.1'
PROTOC_VER = '3.10.0'

# Check host OS
UNAME=`uname -s`

case UNAME.rstrip
  when "Darwin"
    OS="macos"
    SWAGGER_BIN="swagger_darwin_amd64"
    GO_BIN="darwin-amd64.tar.gz"
    PROTOC_ZIP_SUFFIX="osx-x86_64.zip"
  when "Linux"
    OS="linux"
    SWAGGER_BIN="swagger_linux_amd64"
    GO_BIN="linux-amd64.tar.gz"
    PROTOC_ZIP_SUFFIX="linux-x86_64.zip"
  when "FreeBSD"
    OS="FreeBSD"
    SWAGGER_BIN=""
    # Problem here: There are no packages for swagger for FreeBSD
    puts "There are no FreeBSD packages for SWAGGER_BIN"
  else
    puts "Unknown OS: %s" % UNAME
    fail
  end
puts "Detected OS: %s" % OS

# Tool URLs
GOSWAGGER_URL = "https://github.com/go-swagger/go-swagger/releases/download/#{GOSWAGGER_VER}/#{SWAGGER_BIN}"
GOLANGCILINT_URL = "https://github.com/golangci/golangci-lint/releases/download/v#{GOLANGCILINT_VER}/golangci-lint-#{GOLANGCILINT_VER}-linux-amd64.tar.gz"
GO_URL = "https://dl.google.com/go/go#{GO_VER}.#{GO_BIN}"
PROTOC_URL = "https://github.com/protocolbuffers/protobuf/releases/download/v#{PROTOC_VER}/protoc-#{PROTOC_VER}-#{PROTOC_ZIP_SUFFIX}"
PROTOC_GEN_GO_URL = 'github.com/golang/protobuf/protoc-gen-go'
SWAGGER_CODEGEN_URL = "http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/#{SWAGGER_CODEGEN_VER}/swagger-codegen-cli-#{SWAGGER_CODEGEN_VER}.jar"
NODE_URL = "https://nodejs.org/dist/v#{NODE_VER}/node-v#{NODE_VER}-linux-x64.tar.xz"
MOCKERY_URL = 'github.com/vektra/mockery/.../'
MOCKGEN_URL = 'github.com/golang/mock/mockgen'
RICHGO_URL = 'github.com/kyoh86/richgo'

# Tools and Other Paths
TOOLS_DIR = File.expand_path('tools')
NPX = "#{TOOLS_DIR}/node-v#{NODE_VER}-linux-x64/bin/npx"
SWAGGER_CODEGEN = "#{TOOLS_DIR}/swagger-codegen-cli-#{SWAGGER_CODEGEN_VER}.jar"
GOSWAGGER = "#{TOOLS_DIR}/swagger_linux_amd64"
NG = File.expand_path('webui/node_modules/.bin/ng')
GOHOME_DIR = File.expand_path('~/go')
GOBIN = "#{GOHOME_DIR}/bin"
GO = "#{TOOLS_DIR}/go/bin/go"
GOLANGCILINT = "#{TOOLS_DIR}/golangci-lint-#{GOLANGCILINT_VER}-linux-amd64/golangci-lint"
PROTOC = "#{TOOLS_DIR}/protoc/bin/protoc"
PROTOC_GEN_GO = "#{GOBIN}/protoc-gen-go"
MOCKERY = "#{GOBIN}/mockery"
MOCKGEN = "#{GOBIN}/mockgen"
RICHGO = "#{GOBIN}/richgo"

# Patch PATH env
ENV['PATH'] = "#{TOOLS_DIR}/node-v#{NODE_VER}-linux-x64/bin:#{ENV['PATH']}"
ENV['PATH'] = "#{TOOLS_DIR}/go/bin:#{ENV['PATH']}"
ENV['PATH'] = "#{GOBIN}:#{ENV['PATH']}"

# Documentation
SPHINXOPTS = "-v -E -a -W -j 2"

# Files
SWAGGER_FILE = File.expand_path('api/swagger.yaml')
AGENT_PROTO_FILE = File.expand_path('backend/api/agent.proto')
AGENT_PB_GO_FILE = File.expand_path('backend/api/agent.pb.go')

SERVER_GEN_FILES = Rake::FileList[
  File.expand_path('backend/server/gen/restapi/configure_stork.go'),
]

# Directories
directory GOHOME_DIR
directory TOOLS_DIR


# Server Rules
file GO => [TOOLS_DIR, GOHOME_DIR] do
  Dir.chdir(TOOLS_DIR) do
    sh "wget #{GO_URL} -O go.tar.gz"
    sh 'tar -zxf go.tar.gz'
  end
end

file SERVER_GEN_FILES => SWAGGER_FILE do
  Dir.chdir('backend') do
    sh "#{GOSWAGGER} generate server -s server/gen/restapi -m server/gen/models --name Stork --exclude-main --spec #{SWAGGER_FILE} --template stratoscale --regenerate-configureapi"
  end
end

desc 'Generate server part of REST API using goswagger based on swagger.yml'
task :gen_server => [GO, GOSWAGGER, SERVER_GEN_FILES]

file GOSWAGGER => TOOLS_DIR do
  sh "wget #{GOSWAGGER_URL} -O #{GOSWAGGER}"
  sh "chmod a+x #{GOSWAGGER}"
end

desc 'Compile server part'
task :build_server => [GO, :gen_server, :gen_agent] do
  sh 'rm -f backend/server/agentcomm/api_mock.go'
  sh "cd backend/cmd/stork-server/ && #{GO} build"
end

file PROTOC do
  sh "mkdir -p #{TOOLS_DIR}/protoc"
  Dir.chdir("#{TOOLS_DIR}/protoc") do
    sh "wget #{PROTOC_URL} -O protoc.zip"
    sh 'unzip protoc.zip'
  end
end

file PROTOC_GEN_GO do
  sh "#{GO} get -u #{PROTOC_GEN_GO_URL}"
end

file MOCKERY do
  sh "#{GO} get -u #{MOCKERY_URL}"
end

file MOCKGEN do
  sh "#{GO} get -u #{MOCKGEN_URL}"
end

file RICHGO do
  sh "#{GO} get -u #{RICHGO_URL}"
end

file AGENT_PB_GO_FILE => [GO, PROTOC, PROTOC_GEN_GO, AGENT_PROTO_FILE] do
  Dir.chdir('backend') do
    sh "#{PROTOC} -I api api/agent.proto --go_out=plugins=grpc:api"
  end
end

desc 'Generate API sources from agent.proto'
task :gen_agent => [AGENT_PB_GO_FILE]

desc 'Compile agent part'
file :build_agent => [GO, AGENT_PB_GO_FILE] do
  sh "cd backend/cmd/stork-agent/ && #{GO} build"
end

desc 'Run agent'
task :run_agent => [:build_agent, GO] do
  sh "backend/cmd/stork-agent/stork-agent --port 8888"
end


desc 'Run server'
task :run_server, [:dbg] => [:build_server, GO] do |t, args|
  args.with_defaults(:dbg => false)
  if args[:dbg]
    sh "cd backend/cmd/stork-server/ && dlv debug"
  else
    sh "backend/cmd/stork-server/stork-server"
  end
end

desc 'Run server with local postgres docker container'
task :run_server_db, [:dbg] do |t, args|
  args.with_defaults(:dbg => false)
  ENV['STORK_DATABASE_NAME'] = "storkapp"
  ENV['STORK_DATABASE_USER_NAME'] = "storkapp"
  ENV['STORK_DATABASE_PASSWORD'] = "storkapp"
  ENV['STORK_DATABASE_HOST'] = "localhost"
  ENV['STORK_DATABASE_PORT'] = "5678"
  at_exit {
    sh "docker rm -f stork-app-pgsql"
  }
  sh 'docker run --name stork-app-pgsql -d -p 5678:5432 -e POSTGRES_DB=storkapp -e POSTGRES_USER=storkapp -e POSTGRES_PASSWORD=storkapp postgres:11 && sleep 5'
  Rake::Task["run_server"].invoke(args[:dbg])
end


desc 'Compile database migrations tool'
task :build_migrations =>  [GO] do
  sh "cd backend/cmd/stork-db-migrate/ && #{GO} build"
end

desc 'Compile whole backend: server, migrations and agent'
task :build_backend => [:build_agent, :build_server, :build_migrations]

file GOLANGCILINT => TOOLS_DIR do
  Dir.chdir(TOOLS_DIR) do
    sh "wget #{GOLANGCILINT_URL} -O golangci-lint.tar.gz"
    sh "tar -zxf golangci-lint.tar.gz"
  end
end

desc 'Check backend source code'
task :lint_go => [GO, GOLANGCILINT, :gen_server] do
  Dir.chdir('backend/server') do
    sh "#{GOLANGCILINT} run gen/restapi"
  end
end

desc 'Run backend unit tests'
task :unittest_backend => [GO, RICHGO, MOCKERY, MOCKGEN, :build_server, :build_agent] do
  at_exit {
    sh 'rm -f backend/server/agentcomm/api_mock.go'
  }
  sh 'rm -f backend/server/agentcomm/api_mock.go'
  Dir.chdir('backend') do
    sh "#{GO} generate -v ./..."
    sh "#{RICHGO} test -v -p 1 ./..."
  end
end

desc 'Run backend unit tests with local postgres docker container'
task :unittest_backend_db do
  at_exit {
    sh "docker rm -f stork-ut-pgsql"
  }
  sh "docker run --name stork-ut-pgsql -d -p 5678:5432 -e POSTGRES_DB=storktest -e POSTGRES_USER=storktest -e POSTGRES_PASSWORD=storktest postgres:11"
  ENV['POSTGRES_ADDR'] = "localhost:5678"
  Rake::Task["unittest_backend"].invoke
end


# Web UI Rules
desc 'Generate client part of REST API using swagger_codegen based on swagger.yml'
task :gen_client => [SWAGGER_CODEGEN, SWAGGER_FILE] do
  Dir.chdir('webui') do
    sh "java -jar #{SWAGGER_CODEGEN} generate -l typescript-angular -i #{SWAGGER_FILE} -o src/app/backend --additional-properties snapshot=true,ngVersion=8.2.8"
  end
end

file SWAGGER_CODEGEN => TOOLS_DIR do
  sh "wget #{SWAGGER_CODEGEN_URL} -O #{SWAGGER_CODEGEN}"
end

file NPX => TOOLS_DIR do
  Dir.chdir(TOOLS_DIR) do
    sh "wget #{NODE_URL} -O #{TOOLS_DIR}/node.tar.xz"
    sh "tar -Jxf node.tar.xz"
  end
end

file NG => NPX do
  Dir.chdir('webui') do
    sh 'npm install'
  end
end

desc 'Build angular application'
task :build_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx ng build --prod'
  end
end

desc 'Serve angular app'
task :serve_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx ng serve --disable-host-check --proxy-config proxy.conf.json'
  end
end

desc 'Check frontend source code'
task :lint_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx ng lint'
    sh 'npx prettier --config .prettierrc --check \'**/*\''
  end
end

desc 'Make frontend source code prettier'
task :prettier_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx prettier --config .prettierrc --write \'**/*\''
  end
end

# internal task used in ci for running npm ci command with lint and tests together
task :ci_ui => [:gen_client] do
  Dir.chdir('webui') do
    sh 'npm ci'
  end

  Rake::Task["lint_ui"].invoke()

#   Dir.chdir('webui') do
#    sh 'CHROME_BIN=/usr/bin/chromium-browser npx ng test --progress false --watch false'
#    sh 'npx ng e2e --progress false --watch false'
#   end
end


# Docker Rules
desc 'Build containers with everything and statup all services using docker-compose'
task :docker_up => [:build_backend, :build_ui] do
  at_exit {
    sh "docker-compose down"
  }
  sh "docker-compose build"
  sh "docker-compose up"
end

desc 'Shut down all containers'
task :docker_down do
  sh "docker-compose down"
end


# Documentation
desc 'Builds Stork documentation, using Sphinx'
task :docs do
  sh "sphinx-build -M singlehtml docs/ docs/ #{SPHINXOPTS}"
end


# Release Rules
task :tarball do
  version = 'unknown'
  version_file = 'backend/version.go'
  text = File.open(version_file).read
  text.each_line do |line|
    if line.start_with? 'const Version'
      parts = line.split('"')
      version = parts[1]
    end
  end
  sh "git archive --prefix=stork-#{version}/ -o stork-#{version}.tar.gz HEAD"
end


# Other Rules
desc 'Remove tools and other build or generated files'
task :clean do
  sh "rm -rf #{AGENT_PB_GO_FILE}"
  sh 'rm -rf backend/server/gen/*'
  sh 'rm -rf webui/src/app/backend/'
  sh 'rm -f backend/cmd/stork-agent/stork-agent'
  sh 'rm -f backend/cmd/stork-server/stork-server'
  sh 'rm -f backend/cmd/stork-db-migrate/stork-db-migrate'
end

desc 'Download all dependencies'
task :prepare_env => [GO, GOSWAGGER, GOLANGCILINT, SWAGGER_CODEGEN, NPX]
