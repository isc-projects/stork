# coding: utf-8
require 'rake'

# Tool Versions
NODE_VER = '14.18.2'
OPENAPI_GENERATOR_VER = '5.2.0'
GOSWAGGER_VER = 'v0.23.0'
GOLANGCILINT_VER = '1.33.0'
GO_VER = '1.15.5'
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
    NODE_SUFFIX="node-v14.18.2.tar.xz"
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
OPENAPI_GENERATOR_URL = "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/#{OPENAPI_GENERATOR_VER}/openapi-generator-cli-#{OPENAPI_GENERATOR_VER}.jar"
NODE_URL = "https://nodejs.org/dist/v#{NODE_VER}/node-v#{NODE_VER}-#{NODE_SUFFIX}.tar.xz"
MOCKERY_URL = 'github.com/vektra/mockery/.../@v1.0.0'
MOCKGEN_URL = 'github.com/golang/mock/mockgen'
RICHGO_URL = 'github.com/kyoh86/richgo'

# Tools and Other Paths
TOOLS_DIR = File.expand_path('tools')
NPX = "#{TOOLS_DIR}/node-v#{NODE_VER}-#{NODE_SUFFIX}/bin/npx"
OPENAPI_GENERATOR = "#{TOOLS_DIR}/swagger-codegen-cli-#{OPENAPI_GENERATOR_VER}.jar"
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

# wget
if system("wget --version > /dev/null").nil?
  abort("wget is not installed on this system")
end
# extract wget version
WGET_VERSION = `wget --version | head -n 1 | sed -E 's/[^0-9]*([0-9]*\.[0-9]*)[^0-9]+.*/\1/g'`
# versions prior to 1.19 lack support for --retry-on-http-error
if !WGET_VERSION.empty? and WGET_VERSION < "1.19"
  WGET = 'wget --tries=inf --waitretry=3'
else
  WGET = 'wget --tries=inf --waitretry=3 --retry-on-http-error=429,500,503,504 '
end

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
EXAMPLES_GRAFANA_DIR=File.join(EXAMPLES_DIR, 'grafana')
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
directory EXAMPLES_GRAFANA_DIR
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

# CHROME_BIN is required for UI unit tests and system tests. If it is
# not provided by a user, try to locate Chrome binary and set
# environment variable to its location.
if !ENV['CHROME_BIN']
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
      ENV['CHROME_BIN'] = loc
      break
    end
  end
end

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
ut_dbg_headless = ''
if ENV['headless'] == 'true'
  ut_dbg_headless = '--headless -l 0.0.0.0:45678'
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
    sh "cd backend/cmd/stork-agent/ && dlv #{ut_dbg_headless} debug"
  else
    sh "backend/cmd/stork-agent/stork-agent --port 8888"
  end
end

desc 'Run server'
task :run_server => [:build_server, GO] do |t, args|
  ENV['STORK_SERVER_ENABLE_METRICS'] = 'true'
  if ENV['debug'] == 'true'
    sh "cd backend/cmd/stork-server/ && dlv #{ut_dbg_headless} debug"
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


desc 'Compile Stork tool'
task :build_tool =>  [GO] do
  sh "cd backend/cmd/stork-tool/ && #{GO} build #{go_build_date_opt}"
end

desc 'Compile whole backend: server, migrations and agent'
task :build_backend => [:build_agent, :build_server, :build_tool]

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
task :unittest_backend => [GO, RICHGO, MOCKERY, MOCKGEN, :build_server, :build_agent, :build_tool] do
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

  bench_params = ''
  if ENV['benchmark'] == 'true'
    bench_params = '-bench=.'
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

  # exclude long running tests if requested
  short_param = ''
  if ENV['short'] == 'true'
    short_param = '-short'
  end

  # Set the default STORK_DATABASE_PASSWORD for tests.
  if !ENV['STORK_DATABASE_PASSWORD']
    ENV['STORK_DATABASE_PASSWORD'] = 'storktest'
  end

  # prepare database for unit tests: clear any remainings from previous runs, prepare up-to-date template db
  if ENV['POSTGRES_IN_DOCKER'] != 'yes'
    ENV['PGPASSWORD'] = ENV['STORK_DATABASE_PASSWORD']
    remove_remaining_databases(pgsql_host, pgsql_port)
    sh "createdb -h #{pgsql_host} -p #{pgsql_port} -U storktest -O storktest storktest"
  end

  sh "./backend/cmd/stork-tool/stork-tool db-up -d storktest -u storktest --db-host #{pgsql_host} -p #{pgsql_port}"

  if ENV['dbtrace'] == 'true'
    ENV['STORK_DATABASE_TRACE'] = 'true'
  end
  Dir.chdir('backend') do
    sh "#{GO} generate -v ./..."
    if ENV['debug'] == 'true'
      sh "dlv #{ut_dbg_headless} test #{scope}"
    else
      gotool = RICHGO
      if ENV['richgo'] == 'false'
        gotool = GO
      end
      sh "#{gotool} test #{bench_params} #{short_param} -race -v #{cov_params} #{test_regex} #{scope}"  # count=1 disables caching results
    end

    # drop test databases
    if ENV['POSTGRES_IN_DOCKER'] != 'yes'
      ENV['PGPASSWORD'] = ENV['STORK_DATABASE_PASSWORD']
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
                       'ParseArgs', 'NewStorkServer',

                       # this function requires interaction with user so it is hard to test
                       'getAgentAddrAndPortFromUser',

                       # this requires interacting with terminal
                       'GetSecretInTerminal',
                      ]
        if ENV['short'] == 'true'
          ignore_list.concat(['setupRootKeyAndCert', 'setupServerKeyAndCert', 'SetupServerCerts',
                              'ExportSecret'])
        end

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

desc 'Generate client part of REST API using openapi generator based on swagger.yml'
task :gen_client => [OPENAPI_GENERATOR, SWAGGER_FILE] do
  Dir.chdir('webui') do
    sh "java -jar #{OPENAPI_GENERATOR} generate  -g typescript-angular -i #{SWAGGER_FILE} -o src/app/backend --additional-properties snapshot=true,ngVersion=10.1.5,modelPropertyNaming=camelCase"

  end
end

file OPENAPI_GENERATOR => TOOLS_DIR do
  sh "#{WGET} #{OPENAPI_GENERATOR_URL} -O #{OPENAPI_GENERATOR}"
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
    sh 'npx ng build --configuration production'
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
  Rake::Task["ng_test"].invoke()
end

# Common function for running ng test different progress, watch and browsers
# options.
def run_ng_test(progress, watch, browsers)
  if not File.file?(ENV['CHROME_BIN'])
    puts("Chrome binary not found in %s" % [ENV['CHROME_BIN']])
    puts("It is possible to override default Chrome location with CHROME_BIN")
    puts("environment variable. If this variable is set already it seems to")
    puts("point to a wrong location.")
    abort('Aborting tests because Chrome binary was not found.')
  end
   test_opt = ''
   if ENV['test']
    # Globs of test files to include, relative to project root.
    # There are 2 special cases:
    #   when a path to directory is provided, all spec files ending ".spec.@(ts|tsx)" will be included
    #   when a path to a file is provided, and a matching spec file exists it will be included instead
    test_opt = "--include=#{ENV['test']}"
   end
   Dir.chdir('webui') do
     sh "npx ng test #{test_opt} --progress #{progress} --watch #{watch} --browsers=#{browsers}"
#     sh 'npx ng e2e --progress false --watch false'
   end
end

desc 'Run unit tests for Angular with Chrome browser.'
task :ng_test => [NG] do
  if ENV['debug'] == "true" or ENV['headless'] == "false"
    run_ng_test("true", "true", "Chrome")
  else
    run_ng_test("false", "false", "ChromeNoSandboxHeadless")
  end
end

### Docker Tasks #########################

desc 'Build containers with everything and start all services using docker-compose
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

desc 'Build and push demo images'
task :build_and_push_demo_images => :build_all_in_container do
  # build container images with built artifacts
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS}"
  # push built images to docker registry
  sh "docker-compose #{DOCKER_COMPOSE_FILES} push"
end

desc 'Build all in container'
task :build_all_in_container do
  sh 'docker/gen-kea-config.py 7000 > docker/kea-dhcp4-many-subnets.conf'
  # we increase the locked memory limit up to 512kb to work around the problem ocurring on Ubuntu 20.04.
  # for details, see: https://github.com/golang/go/issues/35777 and https://github.com/golang/go/issues/37436
  # The workaround added --ulimit memlock=512 to docker build and --privileged to docker run.
  sh "docker build --ulimit memlock=512 -f docker/docker-builder.txt -t stork-builder ."
  sh "docker run --privileged -v $PWD:/repo --rm stork-builder rake build_all_copy_in_subdir"
end

# internal task used by build_all_in_container
task :build_all_copy_in_subdir do
  sh 'mkdir -p ./build-root'
  sh 'rsync -av --exclude=webui/node_modules --exclude=webui/dist --exclude=webui/src/assets/arm --exclude=webui/src/assets/pkgs --exclude=doc/_build --exclude=doc/doctrees --exclude=backend/server/gen --exclude=*~ --delete api backend doc etc grafana webui Rakefile ./build-root'
  sh "cd ./build-root && GOPATH=/repo/build-root/go rake install_server install_agent"
end

desc 'Build container with Stork Agent and Kea DHCPv4 server'
task :build_kea_container do
  sh 'docker-compose build agent-kea agent-kea-mysql'
end

desc 'Run container with Stork Agent and Kea and mount current Agent binary'
task :run_kea_container do
  at_exit {
    sh 'docker-compose down'
  }
  sh 'docker-compose up --no-deps agent-kea agent-kea-mysql'
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
task :build_kea_premium_container do
  if not ENV['cs_repo_access_token']
    raise 'ERROR: expected cs_repo_access_token to be set'
  end
  if not File.exist?('build-root')
    raise 'ERROR: build-root not found. Run "rake build_all_in_container" first.'
  end
  sh "docker-compose #{DOCKER_COMPOSE_FILES} build #{DOCKER_COMPOSE_PREMIUM_OPTS} agent-kea-premium"
end

desc 'Run container with Stork Agent and Kea with host reseverations in db'
task :run_kea_premium_container do
  at_exit {
    sh "docker-compose -f ./docker-compose.yaml -f ./docker-compose-premium.yaml down"
  }
  sh "docker-compose -f ./docker-compose.yaml -f ./docker-compose-premium.yaml up agent-kea-premium hosts-db"
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

desc 'Generate documentation artifacts from sources'
task :generate_doc_artifacts do
  sh 'drawio doc/src/arch.drawio --export --output doc/static/arch.png'
end

desc 'Update the dependency versions for sphinx'
task :update_python_doc_dependencies do
  sh 'pip-compile -r ./doc/src/requirements.in'
end

### Release Tasks #########################

PKGS_BUILD_DIR = "#{Dir.pwd}/.pkgs-build"
TIMESTAMPED_SRC_TARBALL = "#{PKGS_BUILD_DIR}/stork-#{TIMESTAMP}.tar.gz"

directory PKGS_BUILD_DIR

file TIMESTAMPED_SRC_TARBALL => PKGS_BUILD_DIR do
  sh "rm -f #{PKGS_BUILD_DIR}/stork*.tar.gz"
  sh "git ls-files | tar -czf #{TIMESTAMPED_SRC_TARBALL} -T -"
end

desc 'Prepare release tarball with Stork sources'
task :tarball do
  sh "git archive --prefix=stork-#{STORK_VERSION}/ -o stork-#{STORK_VERSION}.tar.gz HEAD"
end

desc 'Build RPM and deb packages of Stork Agent and Server using Docker'
task :build_pkgs_in_docker => [TIMESTAMPED_SRC_TARBALL, :build_server_pkgs_in_docker]

desc 'Build RPM and deb packages of Stork Server using Docker'
task :build_server_pkgs_in_docker => [TIMESTAMPED_SRC_TARBALL, :build_agent_pkgs_in_docker] do
  run_build_pkg_in_docker('pkgs-ubuntu-18-04', 'deb', 'server')
  run_build_pkg_in_docker('pkgs-centos-8', 'rpm', 'server')
end

desc 'Build RPM and deb packages of Stork Agent using Docker'
task :build_agent_pkgs_in_docker => TIMESTAMPED_SRC_TARBALL do
  run_build_pkg_in_docker('pkgs-ubuntu-18-04', 'deb', 'agent')
  run_build_pkg_in_docker('pkgs-centos-8', 'rpm', 'agent')
end

# Invoke building a package (rpm or deb) of agent or server in given docker image
def run_build_pkg_in_docker(dkr_image, pkg_type, side)
  cmd = "docker run "
  cmd += " -v #{PKGS_BUILD_DIR}:/home/$USER "
  cmd += " -v tools:/tools "
  cmd += " --user `id -u`:`id -g` "
  cmd += " --workdir=\"/home/$USER\""
  cmd += " -e \"HOME=/home/$USER\""
  cmd += " --volume=\"/etc/group:/etc/group:ro\""
  cmd += " --volume=\"/etc/passwd:/etc/passwd:ro\""
  cmd += " --volume=\"/etc/shadow:/etc/shadow:ro\""
  cmd += " --rm"
  cmd += " registry.gitlab.isc.org/isc-projects/stork/#{dkr_image}:latest"
  cmd += " bash -c \""
  cmd +=      " mkdir -p /tmp/build "
  cmd +=      " && tar -C /tmp/build -zxvf /home/$USER/stork-#{TIMESTAMP}.tar.gz"
  cmd +=      " && cp /home/$USER/isc-stork-agent* /tmp/build"
  cmd +=      "  ; cd /tmp/build"
  cmd +=      " && rake build_pkg pkg=#{pkg_type}_#{side} STORK_BUILD_TIMESTAMP=#{TIMESTAMP} GOPATH=/tmp/build/go GOCACHE=/tmp/build/.cache/"
  cmd +=      " && mv isc-stork* /home/$USER"
  cmd +=      " && ls -al /home/$USER/"
  cmd +=      "\""
  sh "#{cmd}"

  puts("Build in docker of #{side}/#{pkg_type} completed.")
  sh "ls -al #{PKGS_BUILD_DIR}/"
  sh "cp #{PKGS_BUILD_DIR}/isc-stork-#{side}*#{pkg_type} ."
  # copy pkgs to web app so it can be served to agent installer
  sh 'mkdir -p webui/src/assets/pkgs'
  sh "rm -f webui/src/assets/pkgs/isc-stork-#{side}*#{pkg_type}"
  sh 'cp isc-stork*deb webui/src/assets/pkgs/'
end

# Internal task that copies sources and builds packages on a side. It is used by run_build_pkg_in_docker.
task :build_pkg do
  # If the host is using an OS other than Linux, e.g. macOS, the appropriate
  # versions of tools will have to be downloaded. Thus, we don't copy the
  # tools from the stork package. If the host OS is Linux, we copy the tools
  # from the package because the Linux specific tools are compatible with
  # the containers onto which they are copied.
  if OS != 'linux' and Dir.exist?("/tools")
    sh "cp -a /tools /tmp/build"
  end
  sh "rm -rf root && rake #{ENV['pkg']} STORK_BUILD_TIMESTAMP=#{TIMESTAMP}"
  sh "ls -al isc-stork*"
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
  sh "cp -a etc/agent-credentials.json.template #{ETC_DIR}"
  sh "cp -a doc/man/stork-agent.8 #{MAN_DIR}"
end

desc 'Install server files to DESTDIR. It depends on building tasks.'
task :install_server => [:build_server, :build_tool, :build_ui, :doc, BIN_DIR, UNIT_DIR, ETC_DIR, WWW_DIR, EXAMPLES_DIR, EXAMPLES_GRAFANA_DIR, MAN_DIR] do
  sh "cp -a backend/cmd/stork-server/stork-server #{BIN_DIR}"
  sh "cp -a backend/cmd/stork-tool/stork-tool #{BIN_DIR}"
  sh "cp -a etc/isc-stork-server.service #{UNIT_DIR}"
  sh "cp -a etc/server.env #{ETC_DIR}"
  sh "cp -a etc/nginx-stork.conf #{EXAMPLES_DIR}"
  sh "cp -a grafana/*json #{EXAMPLES_GRAFANA_DIR}"
  sh "cp -a webui/dist/stork/* #{WWW_DIR}"
  sh "cp -a doc/man/stork-server.8 #{MAN_DIR}"
  sh "cp -a doc/man/stork-tool.8 #{MAN_DIR}"
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
  cmd += " --config-files /etc/stork/#{pkg}.env"
  if pkg == "agent"
    cmd += " --config-files /etc/stork/agent-credentials.json.template"
  end
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
  sh "mkdir -p #{WWW_DIR}/assets/pkgs/"
  # copy pkgs to web app so it can be served to agent installer
  sh "cp -a isc-stork-agent_#{STORK_VERSION}.#{TIMESTAMP}_amd64.deb #{WWW_DIR}/assets/pkgs/"
  sh "cp -a isc-stork-agent-#{STORK_VERSION}.#{TIMESTAMP}-1.x86_64.rpm #{WWW_DIR}/assets/pkgs/"
  fpm('server', 'deb')
end

desc 'Build RPM package with Stork server. It depends on building and installing tasks.'
task :rpm_server => :install_server do
  sh "mkdir -p #{WWW_DIR}/assets/pkgs/"
  # copy pkgs to web app so it can be served to agent installer
  sh "cp -a isc-stork-agent_#{STORK_VERSION}.#{TIMESTAMP}_amd64.deb #{WWW_DIR}/assets/pkgs/"
  sh "cp -a isc-stork-agent-#{STORK_VERSION}.#{TIMESTAMP}-1.x86_64.rpm #{WWW_DIR}/assets/pkgs/"
  fpm('server', 'rpm')
end

desc 'Prepare containers with FPM and other dependencies that are used for building RPM and deb packages'
task :build_fpm_containers do
  sh 'docker build -f docker/pkgs/ubuntu-18-04.txt -t registry.gitlab.isc.org/isc-projects/stork/pkgs-ubuntu-18-04:latest docker/pkgs/'
#  sh 'docker build -f docker/pkgs/centos-8.txt -t registry.gitlab.isc.org/isc-projects/stork/pkgs-centos-8:latest docker/pkgs/'
#  sh 'docker build -f docker/pkgs/cloudsmith.txt -t registry.gitlab.isc.org/isc-projects/stork/pkgs-cloudsmith:latest docker/pkgs/'
end

desc 'Bump up major version'
task :bump_major do
  sh "
    version=$(cat ./api/swagger.in.yaml | grep -E 'version: [0-9]+\.[0-9]+\.[0-9]+' | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+')
    major=$(printf '%s' \"${version}\" | cut -d '.' -f 1)
    new_major=$((major + 1))
    new_version=\"${new_major}.0.0\"
    sed -i \"0,/version: ${version}/s//version: ${new_version}/\" ./api/swagger.in.yaml
    sed -i \"0,/const Version = \\\"${version}\\\"/s//const Version = \\\"${new_version}\\\"/\" ./backend/version.go
    sed -i \"0,/\\\"version\\\": \\\"${version}\\\"/s//\\\"version\\\": \\\"${new_version}\\\"/\" ./webui/package.json
    sed -i \"0,/\\\"version\\\": \\\"${version}\\\"/s//\\\"version\\\": \\\"${new_version}\\\"/\" ./webui/package-lock.json
    printf 'Stork %s released on %s.\n\n' \"${new_version}\" \"$(date -dwednesday +%Y-%m-%d)\" | cat - ./ChangeLog.md > /tmp/stork-changelog && mv /tmp/stork-changelog ./ChangeLog.md
    printf 'Version bumped to %s.\n' \"${new_version}\"

  "
end

desc 'Bump up minor version'
task :bump_minor do
  sh "
    version=$(cat ./api/swagger.in.yaml | grep -E 'version: [0-9]+\.[0-9]+\.[0-9]+' | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+')
    major=$(printf '%s' \"${version}\" | cut -d '.' -f 1)
    minor=$(printf '%s' \"${version}\" | cut -d '.' -f 2)
    new_minor=$((minor + 1))
    new_version=\"${major}.${new_minor}.0\"
    sed -i \"0,/version: ${version}/s//version: ${new_version}/\" ./api/swagger.in.yaml
    sed -i \"0,/const Version = \\\"${version}\\\"/s//const Version = \\\"${new_version}\\\"/\" ./backend/version.go
    sed -i \"0,/\\\"version\\\": \\\"${version}\\\"/s//\\\"version\\\": \\\"${new_version}\\\"/\" ./webui/package.json
    sed -i \"0,/\\\"version\\\": \\\"${version}\\\"/s//\\\"version\\\": \\\"${new_version}\\\"/\" ./webui/package-lock.json
    printf 'Stork %s released on %s.\n\n' \"${new_version}\" \"$(date -dwednesday +%Y-%m-%d)\" | cat - ./ChangeLog.md > /tmp/stork-changelog && mv /tmp/stork-changelog ./ChangeLog.md
    printf 'Version bumped to %s.\n' \"${new_version}\"
  "
end

desc 'Bump up patch version'
task :bump_patch do
  sh "
    version=$(cat ./api/swagger.in.yaml | grep -E 'version: [0-9]+\.[0-9]+\.[0-9]+' | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+')
    major=$(printf '%s' \"${version}\" | cut -d '.' -f 1)
    minor=$(printf '%s' \"${version}\" | cut -d '.' -f 2)
    patch=$(printf '%s' \"${version}\" | cut -d '.' -f 3)
    new_patch=$((patch + 1))
    new_version=\"${major}.${minor}.${new_patch}\"
    sed -i \"0,/version: ${version}/s//version: ${new_version}/\" ./api/swagger.in.yaml
    sed -i \"0,/const Version = \\\"${version}\\\"/s//const Version = \\\"${new_version}\\\"/\" ./backend/version.go
    sed -i \"0,/\\\"version\\\": \\\"${version}\\\"/s//\\\"version\\\": \\\"${new_version}\\\"/\" ./webui/package.json
    sed -i \"0,/\\\"version\\\": \\\"${version}\\\"/s//\\\"version\\\": \\\"${new_version}\\\"/\" ./webui/package-lock.json
    printf 'Stork %s released on %s.\n\n' \"${new_version}\" \"$(date -dwednesday +%Y-%m-%d)\" | cat - ./ChangeLog.md > /tmp/stork-changelog && mv /tmp/stork-changelog ./ChangeLog.md
    printf 'Version bumped to %s.\n' \"${new_version}\"
  "
end


### System testing ######################

PYTEST = './venv/bin/pytest --tb=long -l -r ap -s -v'
SELENIUM_DIR = "#{TOOLS_DIR}/selenium"
directory SELENIUM_DIR

GECKO_DRV_VERSION = '0.28.0'
GECKO_DRV = "#{SELENIUM_DIR}/geckodriver-#{GECKO_DRV_VERSION}"
GECKO_DRV_URL = "https://github.com/mozilla/geckodriver/releases/download/v#{GECKO_DRV_VERSION}/geckodriver-v#{GECKO_DRV_VERSION}-linux64.tar.gz"

if ENV['CHROME_BIN']
  out = `"#{ENV['CHROME_BIN']}" --version`
  if out.include? '85.'
    CHROME_DRV_VERSION = '85.0.4183.87'
  elsif out.include? '86.'
    CHROME_DRV_VERSION = '86.0.4240.22'
  elsif out.include? '87.'
    CHROME_DRV_VERSION = '87.0.4280.20'
  elsif out.include? '90.'
    CHROME_DRV_VERSION = '90.0.4430.72'
  elsif out.include? '92.'
    CHROME_DRV_VERSION = '92.0.4515.159'
  elsif out.include? '93.'
    CHROME_DRV_VERSION = '93.0.4577.63'
  elsif out.include? '94.'
    CHROME_DRV_VERSION = '94.0.4606.61'
  else
    CHROME_DRV_VERSION = ""
    puts "Cannot match Chrome browser version and chromedriver version"
    puts out
  end
  if CHROME_DRV_VERSION
    CHROME_DRV = "#{SELENIUM_DIR}/chromedriver-#{CHROME_DRV_VERSION}"
    CHROME_DRV_URL = "https://chromedriver.storage.googleapis.com/#{CHROME_DRV_VERSION}/chromedriver_linux64.zip"
  end
end

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

desc 'Run system tests exercising REST API and communication with agents'
task :system_tests => 'tests/system/venv/bin/activate' do
  # if provided run particular test
  if ENV['test']
    test = ENV['test']
  else
    test = 'tests.py'
  end

  # use xdist to parallelize tests if requested
  if ENV['xdist']
    xdist_param = "-n #{ENV['xdist']}"
  else
    xdist_param = ''
  end

  # Set single system test timeout
  ENV['PYTEST_TIMEOUT'] = '300'

  # run tests
  Dir.chdir('tests/system') do
    sh './venv/bin/pip install -r requirements.txt'
    sh "#{PYTEST} #{xdist_param} #{test}"
  end
end

file GECKO_DRV => SELENIUM_DIR do
  Dir.chdir(SELENIUM_DIR) do
    sh "#{WGET} #{GECKO_DRV_URL} -O geckodriver.tar.gz"
    sh 'tar -xf geckodriver.tar.gz'
    sh "mv geckodriver #{GECKO_DRV}"
    sh 'rm geckodriver.tar.gz'
    sh "#{GECKO_DRV} -V"
  end
end

if ENV['CHROME_BIN']
  file CHROME_DRV => SELENIUM_DIR do
    Dir.chdir(SELENIUM_DIR) do
      sh "#{WGET} #{CHROME_DRV_URL} -O chromedriver.zip"
      sh "unzip chromedriver.zip"
      sh "mv chromedriver #{CHROME_DRV}"
      sh "rm chromedriver.zip"
      sh "#{CHROME_DRV} --version"
    end
  end
end

desc 'Run web UI system tests. By default Firefox is used.
It can be directly selected by BROWSER variable:
  rake system_tests_ui BROWSER=Firefox
  rake system_tests_ui BROWSER=Chrome'
task :system_tests_ui => ['tests/system/venv/bin/activate', selenium_driver_path] do
  if ENV['test']
    test = ENV['test']
  else
    test = 'ui/tests_ui_basic.py'
  end
  headless_opt = '--headless'
  if ENV['headless'] == "false"
    headless_opt = ''
  end
  Dir.chdir('tests/system') do
    sh './venv/bin/pip install -r requirements.txt'
    sh "#{PYTEST} #{headless_opt} --driver #{ENV['BROWSER']} --driver-path #{selenium_driver_path} #{test}"
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
  sh 'rm -f backend/cmd/stork-tool/stork-tool'
end

desc 'Download all dependencies'
task :prepare_env => [GO, PROTOC, PROTOC_GEN_GO, GOSWAGGER, GOLANGCILINT, OPENAPI_GENERATOR, NPX] do
  Dir.chdir('backend') do
    sh "#{GO} mod download"
  end
  sh "GO111MODULE=on #{GO} get -u github.com/go-delve/delve/cmd/dlv@v1.7.3"
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
