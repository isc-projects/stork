# coding: utf-8
require 'rake'

# Tool Versions
NODE_VER = '12.16.2'
SWAGGER_CODEGEN_VER = '2.4.13'
GOSWAGGER_VER = 'v0.23.0'
GOLANGCILINT_VER = '1.21.0'
GO_VER = '1.13.5'
PROTOC_VER = '3.11.2'
PROTOC_GEN_GO_VER = 'v1.3.3'

# Check host OS
UNAME=`uname -s`

case UNAME.rstrip
  when "Darwin"
    OS="macos"
    GOSWAGGER_BIN="swagger_darwin_amd64"
    GO_SUFFIX="darwin-amd64"
    PROTOC_ZIP_SUFFIX="osx-x86_64"
    NODE_SUFFIX="darwin-x64"
    GOLANGCILINT_SUFFIX="darwin-amd64"
    puts "WARNING: MacOS is not officially supported, the provisions for building on MacOS are made"
    puts "WARNING: for the developers' convenience only."
  when "Linux"
    OS="linux"
    GOSWAGGER_BIN="swagger_linux_amd64"
    GO_SUFFIX="linux-amd64"
    PROTOC_ZIP_SUFFIX="linux-x86_64"
    NODE_SUFFIX="linux-x64"
    GOLANGCILINT_SUFFIX="linux-amd64"
  when "FreeBSD"
    OS="FreeBSD"
    # TODO: there are no swagger built packages for FreeBSD
    GOSWAGGER_BIN=""
    puts "WARNING: There are no FreeBSD packages for GOSWAGGER_BIN"
    GO_SUFFIX="freebsd-amd64"
    # TODO: there are no protoc built packages for FreeBSD (at least as of 3.10.0)
    PROTOC_ZIP_SUFFIX=""
    puts "WARNING: There are no protoc packages built for FreeBSD"
    NODE_SUFFIX="node-v10.16.3.tar.xz"
    GOLANGCILINT_SUFFIX="freebsd-amd64"
  else
    puts "ERROR: Unknown/unsupported OS: %s" % UNAME
    fail
  end

# Tool URLs
GOSWAGGER_URL = "https://github.com/go-swagger/go-swagger/releases/download/#{GOSWAGGER_VER}/#{GOSWAGGER_BIN}"
GOLANGCILINT_URL = "https://github.com/golangci/golangci-lint/releases/download/v#{GOLANGCILINT_VER}/golangci-lint-#{GOLANGCILINT_VER}-#{GOLANGCILINT_SUFFIX}.tar.gz"
GO_URL = "https://dl.google.com/go/go#{GO_VER}.#{GO_SUFFIX}.tar.gz"
PROTOC_URL = "https://github.com/protocolbuffers/protobuf/releases/download/v#{PROTOC_VER}/protoc-#{PROTOC_VER}-#{PROTOC_ZIP_SUFFIX}.zip"
PROTOC_GEN_GO_URL = 'github.com/golang/protobuf/protoc-gen-go'
SWAGGER_CODEGEN_URL = "https://oss.sonatype.org/content/repositories/releases/io/swagger/swagger-codegen-cli/#{SWAGGER_CODEGEN_VER}/swagger-codegen-cli-#{SWAGGER_CODEGEN_VER}.jar"
NODE_URL = "https://nodejs.org/dist/v#{NODE_VER}/node-v#{NODE_VER}-#{NODE_SUFFIX}.tar.xz"
MOCKERY_URL = 'github.com/vektra/mockery/.../@v1.0.0'
MOCKGEN_URL = 'github.com/golang/mock/mockgen'
RICHGO_URL = 'github.com/kyoh86/richgo'

# Tools and Other Paths
TOOLS_DIR = File.expand_path('tools')
NPX = "#{TOOLS_DIR}/node-v#{NODE_VER}-#{NODE_SUFFIX}/bin/npx"
SWAGGER_CODEGEN = "#{TOOLS_DIR}/swagger-codegen-cli-#{SWAGGER_CODEGEN_VER}.jar"
GOSWAGGER_DIR = "#{TOOLS_DIR}/#{GOSWAGGER_VER}"
GOSWAGGER = "#{GOSWAGGER_DIR}/#{GOSWAGGER_BIN}"
NG = File.expand_path('webui/node_modules/.bin/ng')
if ENV['GOPATH']
  GOHOME_DIR = ENV['GOPATH']
else
  GOHOME_DIR = File.expand_path('~/go')
end
GOBIN = "#{GOHOME_DIR}/bin"
GO_DIR = "#{TOOLS_DIR}/#{GO_VER}"
GO = "#{GO_DIR}/go/bin/go"
GOLANGCILINT = "#{TOOLS_DIR}/golangci-lint-#{GOLANGCILINT_VER}-#{GOLANGCILINT_SUFFIX}/golangci-lint"
PROTOC_DIR = "#{TOOLS_DIR}/#{PROTOC_VER}"
PROTOC = "#{PROTOC_DIR}/bin/protoc"
PROTOC_GEN_GO = "#{GOBIN}/protoc-gen-go-#{PROTOC_GEN_GO_VER}"
MOCKERY = "#{GOBIN}/mockery"
MOCKGEN = "#{GOBIN}/mockgen"
RICHGO = "#{GOBIN}/richgo"

WGET = 'wget --tries=inf --waitretry=3 --retry-on-http-error=429,500,503,504 '

# Patch PATH env
ENV['PATH'] = "#{TOOLS_DIR}/node-v#{NODE_VER}-#{NODE_SUFFIX}/bin:#{ENV['PATH']}"
ENV['PATH'] = "#{GO_DIR}/go/bin:#{ENV['PATH']}"
ENV['PATH'] = "#{GOBIN}:#{ENV['PATH']}"

# premium support
if ENV['cs_repo_access_token']
  ENV['premium'] = 'true'
end
if ENV['premium'] == 'true'
  DOCKER_COMPOSE_FILES='-f docker-compose.yaml -f docker-compose-premium.yaml'
  DOCKER_COMPOSE_PREMIUM_OPTS = "--build-arg CS_REPO_ACCESS_TOKEN=#{ENV['cs_repo_access_token']}"
else
  DOCKER_COMPOSE_FILES='-f docker-compose.yaml'
  DOCKER_COMPOSE_PREMIUM_OPTS = ''
end

# build date
now = Time.now
build_date = now.strftime("%Y-%m-%d %H:%M")
if ENV['STORK_BUILD_TIMESTAMP']
  TIMESTAMP = ENV['STORK_BUILD_TIMESTAMP']
else
  TIMESTAMP = now.strftime("%y%m%d%H%M%S")
end
puts "Stork build date: #{build_date} (timestamp: #{TIMESTAMP})"
go_build_date_opt = "-ldflags=\"-X 'isc.org/stork.BuildDate=#{build_date}'\""

# Documentation
SPHINXOPTS = "-v -E -a -W -j 2"

# Files
SWAGGER_FILE = File.expand_path('api/swagger.yaml')
SWAGGER_API_FILES = [
  'api/swagger.in.yaml',
  'api/services-defs.yaml', 'api/services-paths.yaml',
  'api/users-defs.yaml', 'api/users-paths.yaml',
  'api/dhcp-defs.yaml', 'api/dhcp-paths.yaml',
  'api/settings-defs.yaml', 'api/settings-paths.yaml',
  'api/search-defs.yaml', 'api/search-paths.yaml',
  'api/events-defs.yaml', 'api/events-paths.yaml'
]
AGENT_PROTO_FILE = File.expand_path('backend/api/agent.proto')
AGENT_PB_GO_FILE = File.expand_path('backend/api/agent.pb.go')

SERVER_GEN_FILES = Rake::FileList[
  File.expand_path('backend/server/gen/restapi/configure_stork.go'),
]

# locations for installing files
if ENV['DESTDIR']
  DESTDIR=ENV['DESTDIR']
else
  DESTDIR=File.join(File.dirname(__FILE__), 'root')
end
BIN_DIR=File.join(DESTDIR, 'usr/bin')
UNIT_DIR=File.join(DESTDIR, 'lib/systemd/system')
ETC_DIR=File.join(DESTDIR, 'etc/stork')
WWW_DIR=File.join(DESTDIR, 'usr/share/stork/www')
EXAMPLES_DIR=File.join(DESTDIR, 'usr/share/stork/examples')
MAN_DIR=File.join(DESTDIR, 'usr/share/man/man8')

# Directories
directory GOHOME_DIR
directory TOOLS_DIR
directory DESTDIR
directory BIN_DIR
directory UNIT_DIR
directory ETC_DIR
directory WWW_DIR
directory EXAMPLES_DIR
directory MAN_DIR

# establish Stork version
stork_version = '0.0.0'
version_file = 'backend/version.go'
text = File.open(version_file).read
text.each_line do |line|
  if line.start_with? 'const Version'
    parts = line.split('"')
    stork_version = parts[1]
  end
end
STORK_VERSION = stork_version

### Backend Tasks #########################

file GO => [TOOLS_DIR, GOHOME_DIR] do
  sh "mkdir -p #{GO_DIR}"
  sh "#{WGET} #{GO_URL} -O #{GO_DIR}/go.tar.gz"
  Dir.chdir(GO_DIR) do
    sh 'tar -zxf go.tar.gz'
  end
end

YAMLINC = File.expand_path('webui/node_modules/.bin/yamlinc')

file YAMLINC do
  Rake::Task[NG].invoke()
end

file SWAGGER_FILE => [YAMLINC, *SWAGGER_API_FILES] do
  Dir.chdir('api') do
    sh "#{YAMLINC} -o swagger.yaml swagger.in.yaml"
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
  sh "mkdir -p #{GOSWAGGER_DIR}"
  sh "#{WGET} #{GOSWAGGER_URL} -O #{GOSWAGGER}"
  sh "chmod a+x #{GOSWAGGER}"
end

desc 'Compile server part'
task :build_server => [GO, :gen_server, :gen_agent] do
  sh 'rm -f backend/server/agentcomm/api_mock.go'
  sh "cd backend/cmd/stork-server/ && #{GO} build #{go_build_date_opt}"
end

file PROTOC do
  sh "mkdir -p #{PROTOC_DIR}"
  sh "#{WGET} #{PROTOC_URL} -O #{PROTOC_DIR}/protoc.zip"
  Dir.chdir(PROTOC_DIR) do
    sh 'unzip protoc.zip'
  end
end

file PROTOC_GEN_GO do
  sh "#{GO} get -d -u #{PROTOC_GEN_GO_URL}"
  sh "git -C \"$(#{GO} env GOPATH)\"/src/github.com/golang/protobuf checkout #{PROTOC_GEN_GO_VER}"
  sh "#{GO} install github.com/golang/protobuf/protoc-gen-go"
  sh "cp #{GOBIN}/protoc-gen-go #{PROTOC_GEN_GO}"
end

file MOCKERY do
  Dir.chdir('backend') do
    sh "#{GO} get #{MOCKERY_URL}"
  end
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

# prepare args for dlv debugger
headless = ''
if ENV['headless'] == 'true'
  headless = '--headless -l 0.0.0.0:45678'
end

desc 'Connect gdlv GUI Go debugger to waiting dlv debugger'
task :connect_dbg do
  sh 'gdlv connect 127.0.0.1:45678'
end

desc 'Generate API sources from agent.proto'
task :gen_agent => [AGENT_PB_GO_FILE]

desc 'Compile agent part'
file :build_agent => [GO, AGENT_PB_GO_FILE] do
  sh "cd backend/cmd/stork-agent/ && #{GO} build #{go_build_date_opt}"
end

desc 'Run agent'
task :run_agent => [:build_agent, GO] do
  if ENV['debug'] == 'true'
    sh "cd backend/cmd/stork-agent/ && dlv #{headless} debug"
  else
    sh "backend/cmd/stork-agent/stork-agent --port 8888"
  end
end

desc 'Run server'
task :run_server => [:build_server, GO] do |t, args|
  if ENV['debug'] == 'true'
    sh "cd backend/cmd/stork-server/ && dlv #{headless} debug"
  else
    cmd = 'backend/cmd/stork-server/stork-server'
    if ENV['dbtrace'] == 'true'
      cmd = "#{cmd} --db-trace-queries=run"
    end
    sh cmd
  end
end

desc 'Run server with local postgres docker container'
task :run_server_db do |t, args|
  ENV['STORK_DATABASE_NAME'] = "storkapp"
  ENV['STORK_DATABASE_USER_NAME'] = "storkapp"
  ENV['STORK_DATABASE_PASSWORD'] = "storkapp"
  ENV['STORK_DATABASE_HOST'] = "localhost"
  ENV['STORK_DATABASE_PORT'] = "5678"
  at_exit {
    sh "docker rm -f -v stork-app-pgsql"
  }
  sh 'docker run --name stork-app-pgsql -d -p 5678:5432 -e POSTGRES_DB=storkapp -e POSTGRES_USER=storkapp -e POSTGRES_PASSWORD=storkapp postgres:11 && sleep 5'
  Rake::Task["run_server"].invoke()
end


desc 'Compile database migrations tool'
task :build_migrations =>  [GO] do
  sh "cd backend/cmd/stork-db-migrate/ && #{GO} build #{go_build_date_opt}"
end

desc 'Compile whole backend: server, migrations and agent'
task :build_backend => [:build_agent, :build_server, :build_migrations]

file GOLANGCILINT => TOOLS_DIR do
  Dir.chdir(TOOLS_DIR) do
    sh "#{WGET} #{GOLANGCILINT_URL} -O golangci-lint.tar.gz"
    sh "tar -zxf golangci-lint.tar.gz"
  end
end

desc 'Build Stork backend continuously whenever source files change'
task :backend_live do
  sh 'find backend -name "*.go" | entr rake build_backend'
end

desc 'Check backend source code
arguments: fix=true - fixes some of the found issues'
task :lint_go => [GO, GOLANGCILINT, MOCKERY, MOCKGEN, :gen_agent, :gen_server] do
  at_exit {
    sh 'rm -f backend/server/agentcomm/api_mock.go'
  }
  sh 'rm -f backend/server/agentcomm/api_mock.go'
  Dir.chdir('backend') do
    sh "#{GO} generate -v ./..."

    opts = ''
    if ENV['fix'] == 'true'
      opts += ' --fix'
    end
    sh "#{GOLANGCILINT} run #{opts}"
  end
end

desc 'Format backend source code'
task :fmt_go => [GO, :gen_agent, :gen_server] do
  Dir.chdir('backend') do
    sh "#{GO} fmt ./..."
  end
end

def remove_remaining_databases(pgsql_host, pgsql_port)
    sh %{
       for db in $(psql -t -h #{pgsql_host} -p #{pgsql_port} -U storktest -c \"select datname from pg_database where datname ~ 'storktest.*'\"); do
         dropdb -h #{pgsql_host} -p #{pgsql_port} -U storktest $db
       done
    }
end

desc 'Run backend unit tests'
task :unittest_backend => [GO, RICHGO, MOCKERY, MOCKGEN, :build_server, :build_agent, :build_migrations] do
  at_exit {
    sh 'rm -f backend/server/agentcomm/api_mock.go'
  }
  sh 'rm -f backend/server/agentcomm/api_mock.go'

  cov_params = '-coverprofile=coverage.out'

  if ENV['scope']
    scope = ENV['scope']
    cov_params = ''
  else
    scope = './...'
  end

  if ENV['test']
    test_regex = "-run #{ENV['test']}"
    cov_params = ''
  else
    test_regex = ''
  end

  # establish location of postgresql database
  pgsql_host = 'localhost'
  pgsql_port = '5432'
  if ENV['POSTGRES_ADDR']
    pgsql_addr = ENV['POSTGRES_ADDR']
    if pgsql_addr.include? ':'
      pgsql_host, pgsql_port = pgsql_addr.split(':')
    else
      pgsql_host = pgsql_addr
    end
  end

  # prepare database for unit tests: clear any remainings from previous runs, prepare up-to-date template db
  if ENV['POSTGRES_IN_DOCKER'] != 'yes'
    remove_remaining_databases(pgsql_host, pgsql_port)
    sh "createdb -h #{pgsql_host} -p #{pgsql_port} -U storktest -O storktest storktest"
  end
  sh "STORK_DATABASE_PASSWORD=storktest ./backend/cmd/stork-db-migrate/stork-db-migrate -d storktest -u storktest --db-host #{pgsql_host} -p #{pgsql_port} up"

  if ENV['dbtrace'] == 'true'
    ENV['STORK_DATABASE_TRACE'] = 'true'
  end
  Dir.chdir('backend') do
    sh "#{GO} generate -v ./..."
    if ENV['debug'] == 'true'
      sh "dlv #{headless} test #{scope}"
    else
      gotool = RICHGO
      if ENV['richgo'] == 'false'
        gotool = GO
      end
      sh "#{gotool} test -race -v #{cov_params} #{test_regex} #{scope}"  # count=1 disables caching results
    end

    # drop test databases
    if ENV['POSTGRES_IN_DOCKER'] != 'yes'
      remove_remaining_databases(pgsql_host, pgsql_port)
    end

    # check coverage level (run it only for full tests scope)
    if not ENV['scope'] and not ENV['test']
      out = `#{GO} tool cover -func=coverage.out`
      puts out, ''
      problem = false
      out.each_line do |line|
        if line.start_with? 'total:' or line.include? 'api_mock.go'
          next
        end
        items = line.gsub(/\s+/m, ' ').strip.split(" ")
        file = items[0]
        func = items[1]
        cov = items[2].strip()[0..-2].to_f
        ignore_list = ['DetectServices', 'RestartKea', 'Serve', 'BeforeQuery', 'AfterQuery',
                       'Identity', 'LogoutHandler', 'NewDatabaseSettings', 'ConnectionParams',
                       'Password', 'loggingMiddleware', 'GlobalMiddleware', 'Authorizer',
                       'Listen', 'Shutdown', 'SetupLogging', 'UTCNow', 'detectApps',
                       'prepareTLS', 'handleRequest', 'pullerLoop', 'Output', 'Collect',
                       'collectTime', 'collectResolverStat', 'collectResolverLabelStat',

                       # Those two are tested in backend/server/server_test.go, in TestCommandLineSwitches*
                       # However, due to how it's executed (calling external binary), it's not detected
                       # by coverage.
                       'ParseArgs', 'NewStorkServer']
        if cov < 35 and not ignore_list.include? func
          puts "FAIL: %-80s %5s%% < 35%%" % ["#{file} #{func}", "#{cov}"]
          problem = true
        end
      end
      if problem
        fail("\nFAIL: Tests coverage is too low, add some tests\n\n")
      end
    end
  end
end

desc 'Run backend unit tests with local postgres docker container'
task :unittest_backend_db do
  at_exit {
    sh "docker rm -f -v stork-ut-pgsql"
  }
  sh "docker run --name stork-ut-pgsql -d -p 5678:5432 -e POSTGRES_DB=storktest -e POSTGRES_USER=storktest -e POSTGRES_PASSWORD=storktest postgres:11"
  ENV['POSTGRES_ADDR'] = "localhost:5678"
  ENV['POSTGRES_IN_DOCKER'] = 'yes'
  Rake::Task["unittest_backend"].invoke
end

desc 'Show backend coverage of unit tests in web browser'
task :show_cov do
  at_exit {
    sh 'rm -f backend/server/agentcomm/api_mock.go'
  }
  if not File.file?('backend/coverage.out')
    Rake::Task["unittest_backend_db"].invoke()
  end
  Dir.chdir('backend') do
    sh "#{GO} generate -v ./..."
    sh "#{GO} tool cover -html=coverage.out"
  end
end


### Web UI Tasks #########################

desc 'Generate client part of REST API using swagger_codegen based on swagger.yml'
task :gen_client => [SWAGGER_CODEGEN, SWAGGER_FILE] do
  Dir.chdir('webui') do
    sh "java -jar #{SWAGGER_CODEGEN} generate -l typescript-angular -i #{SWAGGER_FILE} -o src/app/backend --additional-properties snapshot=true,ngVersion=8.2.8"
  end
end

file SWAGGER_CODEGEN => TOOLS_DIR do
  sh "#{WGET} #{SWAGGER_CODEGEN_URL} -O #{SWAGGER_CODEGEN}"
end

file NPX => TOOLS_DIR do
  Dir.chdir(TOOLS_DIR) do
    sh "#{WGET} #{NODE_URL} -O #{TOOLS_DIR}/node.tar.xz"
    sh "tar -Jxf node.tar.xz"
  end
end

file NG => NPX do
  Dir.chdir('webui') do
    sh 'NG_CLI_ANALYTICS=false env -u DESTDIR npm install'
  end
end

desc 'Build angular application'
task :build_ui => [NG, :gen_client, :doc] do
  Dir.chdir('webui') do
    sh 'npx ng build --prod'
  end
end

desc 'Build Stork Angular app continuously whenever source files change'
task :serve_ui => [NG, :gen_client, :doc] do
  Dir.chdir('webui') do
    sh 'npx ng build --watch'
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


### Docker Tasks #########################

desc 'Build containers with everything and statup all services using docker-compose
arguments: cache=false - forces rebuilding whole container'
task :docker_up => :build_all_in_container do
  at_exit {
    sh "docker-compose #{DOCKER_COMPOSE_FILES} down"
  }
  cache_opt = ''
  if ENV['cache'] == 'false'
    cache_opt = '--no-cache'
  end
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS} #{cache_opt}"
  sh "docker-compose #{DOCKER_COMPOSE_FILES} up"
end

desc 'Shut down all containers'
task :docker_down do
  sh "docker-compose #{DOCKER_COMPOSE_FILES} down"
end

desc 'Build all in container'
task :build_all_in_container do
  # we increase the locked memory limit up to 512kb to work around the problem ocurring on Ubuntu 20.04.
  # for details, see: https://github.com/golang/go/issues/35777 and https://github.com/golang/go/issues/37436
  # The workaround added --ulimit memlock=512 to docker build and --privileged to docker run.
  sh "docker build --ulimit memlock=512 -f docker/docker-builder.txt -t stork-builder ."
  sh "docker run --privileged -v $PWD:/repo --rm stork-builder rake build_all_copy_in_subdir"
end

# internal task used by build_all_in_container
task :build_all_copy_in_subdir do
  sh 'mkdir -p ./build-root'
  sh 'rsync -av --exclude=webui/node_modules --exclude=webui/dist --exclude=webui/src/assets/arm --exclude=doc/_build --exclude=doc/doctrees --exclude=backend/server/gen --exclude=*~ --delete api backend doc etc webui Rakefile ./build-root'
  sh "cd ./build-root && GOPATH=/repo/build-root/go rake install_server install_agent"
end

desc 'Build container with Stork Agent and Kea DHCPv4 server'
task :build_kea_container do
  sh 'docker-compose build agent-kea'
end

desc 'Run container with Stork Agent and Kea and mount current Agent binary'
task :run_kea_container do
  at_exit {
    sh 'docker-compose down'
  }
  sh 'docker-compose up agent-kea'
end

desc 'Build container with Stork Agent and Kea DHCPv6 server'
task :build_kea6_container do
  sh 'docker-compose build agent-kea6'
end

desc 'Run container with Stork Agent and Kea DHCPv6 server and mount current Agent binary'
task :run_kea6_container do
  at_exit {
    sh 'docker-compose down'
  }
  sh 'docker-compose up agent-kea6'
end

desc 'Build two containers with Stork Agent and Kea HA pair
arguments: cache=false - forces rebuilding whole container'
task :build_kea_ha_containers do
  cache_opt = ''
  if ENV['cache'] == 'false'
    cache_opt = '--no-cache'
  end
  sh "docker-compose build #{cache_opt} agent-kea-ha1 agent-kea-ha2"
end

desc 'Run two containers with Stork Agent and Kea HA pair'
task :run_kea_ha_containers do
  at_exit {
    sh "docker-compose down"
  }
  sh 'docker-compose up agent-kea-ha1 agent-kea-ha2'
end

desc 'Build container with Stork Agent and Kea with host reseverations in db'
task :build_kea_hosts_container do
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS} agent-kea-hosts"
end

desc 'Run container with Stork Agent and Kea with host reseverations in db'
task :run_kea_hosts_container do
  at_exit {
    sh "docker-compose #{DOCKER_COMPOSE_FILES} down"
  }
  sh "docker-compose #{DOCKER_COMPOSE_FILES} up agent-kea-hosts hosts-db"
end

desc 'Build container with Stork Agent and BIND 9'
task :build_bind9_container do
  sh 'docker build -f docker/docker-agent-bind9.txt -t agent-bind9 .'
end

desc 'Run container with Stork Agent and BIND 9 and mount current Agent binary'
task :run_bind9_container do
  # host[9999]->agent[8080]
  sh 'docker run --rm -ti -p 9999:8080 --name agent-bind9 -h agent-bind9 -v `pwd`/backend/cmd/stork-agent:/agent agent-bind9'
end

file 'tests/sim/venv/bin/activate' do
  Dir.chdir('tests/sim') do
    sh 'python3 -m venv venv'
    sh './venv/bin/pip install -U pip'
  end
end

file 'tests/sim/venv/bin/flask' => 'tests/sim/venv/bin/activate' do
  Dir.chdir('tests/sim') do
    sh './venv/bin/pip install -r requirements.txt'
  end
end

desc 'Run simulator for experimenting with Stork'
task :run_sim => 'tests/sim/venv/bin/flask' do
  Dir.chdir('tests/sim') do
    sh 'STORK_SERVER_URL=http://localhost:8080 FLASK_ENV=development FLASK_APP=sim.py LC_ALL=C.UTF-8 LANG=C.UTF-8 ./venv/bin/flask run --host 0.0.0.0 --port 5005'
  end
end

### Documentation Tasks #########################

desc 'Builds Stork documentation, using Sphinx'
task :doc do
  sh "sphinx-build -M html doc/ doc/_build #{SPHINXOPTS}"
  sh 'mkdir -p webui/src/assets/arm'
  sh 'cp -a doc/_build/html/* webui/src/assets/arm'
  sh "sphinx-build -M man doc/ doc/ #{SPHINXOPTS}"
end

desc 'Builds Stork documentation continuously whenever source files change'
task :doc_live do
  sh 'find doc -name "*.rst" | entr rake doc'
end


### Release Tasks #########################

desc 'Prepare release tarball with Stork sources'
task :tarball do
  sh "git archive --prefix=stork-#{STORK_VERSION}/ -o stork-#{STORK_VERSION}.tar.gz HEAD"
end

desc 'Build debs in Docker. It is used for developer purposes.'
task :build_debs_in_docker do
  sh "docker run -v $PWD:/repo --rm -ti registry.gitlab.isc.org/isc-projects/stork/pkgs-ubuntu-18-04:latest rake build_pkgs STORK_BUILD_TIMESTAMP=#{TIMESTAMP}"
end

desc 'Build RPMs in Docker. It is used for developer purposes.'
task :build_rpms_in_docker do
  sh "docker run -v $PWD:/repo --rm -ti registry.gitlab.isc.org/isc-projects/stork/pkgs-centos-8:latest rake build_pkgs STORK_BUILD_TIMESTAMP=#{TIMESTAMP}"
end

task :build_pkgs_in_docker => [:build_debs_in_docker, :build_rpms_in_docker]

# Internal task that copies sources and builds packages on a side. It is used by build_debs_in_docker and build_rpms_in_docker.
task :build_pkgs do
  sh 'rm -rf /build && mkdir /build'
  sh 'git archive -o /stork.tar.gz HEAD'
  sh 'tar -C /build -zxvf /stork.tar.gz'
  cwd = Dir.pwd
  if Dir.exist?("#{cwd}/tools")
    sh "cp -a #{cwd}/tools /build"
  end
  if File.exist?('/etc/redhat-release')
    pkg_type = 'rpm'
  else
    pkg_type = 'deb'
  end
  sh "cd /build && rm -rf /build/root && rake #{pkg_type}_agent STORK_BUILD_TIMESTAMP=#{TIMESTAMP}"
  sh "cd /build && rm -rf /build/root && rake #{pkg_type}_server STORK_BUILD_TIMESTAMP=#{TIMESTAMP}"
  sh "cp /build/isc-stork* #{cwd}"
  sh "ls -al #{cwd}/isc-stork*"
end

desc 'Build all. It builds backend and UI.'
task :build_all => [:build_backend, :build_ui] do
  sh 'echo DONE'
end

desc 'Install agent files to DESTDIR. It depends on building tasks.'
task :install_agent => [:build_agent, :doc, BIN_DIR, UNIT_DIR, ETC_DIR, MAN_DIR] do
  sh "cp -a backend/cmd/stork-agent/stork-agent #{BIN_DIR}"
  sh "cp -a etc/isc-stork-agent.service #{UNIT_DIR}"
  sh "cp -a etc/agent.env #{ETC_DIR}"
  sh "cp -a doc/man/stork-agent.8 #{MAN_DIR}"
end

desc 'Install server files to DESTDIR. It depends on building tasks.'
task :install_server => [:build_server, :build_migrations, :build_ui, :doc, BIN_DIR, UNIT_DIR, ETC_DIR, WWW_DIR, EXAMPLES_DIR, MAN_DIR] do
  sh "cp -a backend/cmd/stork-server/stork-server #{BIN_DIR}"
  sh "cp -a backend/cmd/stork-db-migrate/stork-db-migrate #{BIN_DIR}"
  sh "cp -a etc/isc-stork-server.service #{UNIT_DIR}"
  sh "cp -a etc/server.env #{ETC_DIR}"
  sh "cp -a etc/nginx-stork.conf #{EXAMPLES_DIR}"
  sh "cp -a webui/dist/stork/* #{WWW_DIR}"
  sh "cp -a doc/man/stork-server.8 #{MAN_DIR}"
  sh "cp -a doc/man/stork-db-migrate.8 #{MAN_DIR}"
end

# invoke fpm for building RPM or deb package
def fpm(pkg, fpm_target)
  cmd = "fpm -n isc-stork-#{pkg}"
  cmd += " -v #{STORK_VERSION}.#{TIMESTAMP}"
  cmd += " --license 'MPL 2.0'"
  cmd += " --vendor 'Internet Systems Consortium, Inc.'"
  cmd += " --url 'https://gitlab.isc.org/isc-projects/stork/'"
  cmd += " --description 'ISC Stork #{pkg.capitalize()}'"
  cmd += " --after-install etc/isc-stork-#{pkg}.postinst"
  cmd += " --before-remove etc/isc-stork-#{pkg}.prerm"
  cmd += " --after-remove etc/isc-stork-#{pkg}.postrm"
  cmd += " -s dir"
  cmd += " -t #{fpm_target}"
  cmd += " -C #{DESTDIR} ."
  sh cmd
end

desc 'Build deb package with Stork agent. It depends on building and installing tasks.'
task :deb_agent => :install_agent do
  fpm('agent', 'deb')
end

desc 'Build RPM package with Stork agent. It depends on building and installing tasks.'
task :rpm_agent => :install_agent do
  fpm('agent', 'rpm')
end

desc 'Build deb package with Stork server. It depends on building and installing tasks.'
task :deb_server => :install_server do
  fpm('server', 'deb')
end

desc 'Build RPM package with Stork server. It depends on building and installing tasks.'
task :rpm_server => :install_server do
  fpm('server', 'rpm')
end

desc 'Prepare containers with FPM and other dependencies that are used for building RPM and deb packages'
task :build_fpm_containers do
#  sh 'docker build -f docker/pkgs/ubuntu-18-04.txt -t registry.gitlab.isc.org/isc-projects/stork/pkgs-ubuntu-18-04:latest docker/pkgs/'
#  sh 'docker build -f docker/pkgs/centos-8.txt -t registry.gitlab.isc.org/isc-projects/stork/pkgs-centos-8:latest docker/pkgs/'
  sh 'docker build -f docker/pkgs/cloudsmith.txt -t registry.gitlab.isc.org/isc-projects/stork/pkgs-cloudsmith:latest docker/pkgs/'
end


### System testing ######################

SELENIUM_DIR = "#{TOOLS_DIR}/selenium"
GECKO_DRV = "#{SELENIUM_DIR}/geckodriver"
CHROME_DRV = "#{SELENIUM_DIR}/chromedriver"
directory SELENIUM_DIR

if ENV['BROWSER'] == 'Chrome'
  selenium_driver_path = CHROME_DRV
else
  ENV['BROWSER'] = 'Firefox'
  selenium_driver_path = GECKO_DRV
end


file 'tests/system/venv/bin/activate' do
  Dir.chdir('tests/system') do
    sh 'python3 -m venv venv'
    sh './venv/bin/pip install -U pip'
  end
end

task :system_tests => 'tests/system/venv/bin/activate' do
  Dir.chdir('tests/system') do
    sh './venv/bin/pip install -r requirements.txt'
    sh './venv/bin/pytest --full-trace -r ap -s tests.py'
  end
end

file GECKO_DRV => SELENIUM_DIR do
  Dir.chdir(SELENIUM_DIR) do
    sh "#{WGET} https://github.com/mozilla/geckodriver/releases/download/v0.26.0/geckodriver-v0.26.0-linux32.tar.gz -O geckodriver.tar.gz"
    sh 'tar -xf geckodriver.tar.gz'
    sh 'rm geckodriver.tar.gz'
  end
end

file CHROME_DRV => SELENIUM_DIR do
  Dir.chdir(SELENIUM_DIR) do
  sh "#{WGET} https://chromedriver.storage.googleapis.com/85.0.4183.87/chromedriver_linux64.zip -O chromedriver_linux64.zip"
    sh "unzip chromedriver_linux64.zip"
    sh "rm chromedriver_linux64.zip"
  end
end

desc 'Run web UI system tests. By default Firefox is used.
It can be directly selected by BROWSER variable:
  rake system_tests_ui BROWSER=Firefox
  rake system_tests_ui BROWSER=Chrome'
task :system_tests_ui => ['tests/system/venv/bin/activate', selenium_driver_path] do
  Dir.chdir('tests/system') do
    sh './venv/bin/pip install -r requirements.txt'
    sh "./venv/bin/pytest --driver #{ENV['BROWSER']} --driver-path #{selenium_driver_path} -vv --full-trace -r ap -s tests/ui/tests_ui_basic.py --headless"
  end
end



### Other Tasks #########################

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
task :prepare_env => [GO, PROTOC, PROTOC_GEN_GO, GOSWAGGER, GOLANGCILINT, SWAGGER_CODEGEN, NPX] do
  Dir.chdir('backend') do
    sh "#{GO} mod download"
  end
  sh "#{GO} get -u github.com/go-delve/delve/cmd/dlv"
  sh "#{GO} get -u github.com/aarzilli/gdlv"
end

desc 'Generate ctags for Emacs'
task :ctags do
  sh 'etags.ctags -f TAGS -R --exclude=webui/node_modules --exclude=webui/dist --exclude=tools .'
end

desc 'Prepare containers that are using in GitLab CI processes'
task :build_ci_containers do
  sh 'docker build --no-cache -f docker/docker-ci-base.txt -t registry.gitlab.isc.org/isc-projects/stork/ci-base:latest docker/'
  #sh 'docker push registry.gitlab.isc.org/isc-projects/stork/ci-base:latest'
end
