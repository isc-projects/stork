###############
### Backend ###
###############

go_codebase = GO_SERVER_CODEBASE +
        GO_AGENT_CODEBASE +
        GO_TOOL_CODEBASE

go_dev_codebase = go_codebase + [GO_SERVER_API_MOCK]

desc 'Build Stork backend continuously whenever source files change'
task :build_backend_live => go_codebase do
    Open3.pipeline(
        ['printf', '%s\\n', *go_codebase],
        ['entr', '-d', 'rake', 'build_backend']
    )
end

desc 'Check backend source code'
task :lint_go, [:fix] => [GOLANGCILINT] + go_dev_codebase do |t, args|
    args.with_defaults(:fix => "false")
    opts = []
    if args.fix == "true"
        opts += ["--fix"]
    end

    Dir.chdir("backend") do
        sh GOLANGCILINT, "run", *opts
    end
end

desc 'Format backend source code'
task :fmt_go => [GO] + go_codebase do
    Dir.chdir('backend') do
        sh GO, "fmt", "./..."
    end
end

desc 'Run backend unit and coverage tests'
task :unittest_backend, [:scope, :test, :benchmark, :short,
        :dbhost, :dbport, :dbpass, :dbtrace] => [RICHGO, :db_remove_remaining, :db_migrate] + go_dev_codebase do |t, args|
    args.with_defaults(
        :scope => "./...",
        :benchmark => "false",
        :short => "false"
    )

    opts = []

    if args.test != nil
        opts += ["-run", args.test]
    end

    if args.benchmark == "true"
        opts += ["-bench=."]
    end

    if args.short == "true"
        opts += ["-short"]
    end

    with_cov_tests = args.scope == "./..." && args.test == nil

    if with_cov_tests
        opts += ["-coverprofile=coverage.out"]

        at_exit {
            sh "rm -f backend/coverage.out"
        }
    end

    Dir.chdir('backend') do
        sh RICHGO, "test", *opts, "-race", "-v", args.scope

        if with_cov_tests
            out = `"#{GO}" tool cover -func=coverage.out`
        
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
                if args.short == 'true'
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

desc 'Show backend coverage of unit tests in web browser'
task :show_backend_cov, [:dbhost, :dbport, :dbpass, :dbtrace] => [GO, :unittest_backend] do
    puts "Warning: Coverage may not work under Chrome-like browsers; use Firefox if any problems occur."
    Dir.chdir('backend') do
        sh GO, "tool", "cover", "-html=coverage.out"
    end
end

##########################
### Backend- Debugging ###
##########################

desc 'Connect gdlv GUI Go debugger to waiting dlv debugger'
task :connect_dbg => GDLV do
  sh GDLV, "connect", "127.0.0.1:45678"
end

desc 'Run backend unit tests (debug mode)'
task :unittest_backend_debug, [:scope, :short, :headless,
        :dbhost, :dbport, :dbpass, :dbtrace] => [DLV, :db_remove_remaining, :db_migrate] + go_dev_codebase do |t, args|

    if args.scope == nil
        puts "Scope argument is required"
        fail
    end

    opts = []

    if args.headless == "true"
        opts = ["--headless", "-l", "0.0.0.0:45678"]
    end

    Dir.chdir('backend') do
        sh DLV, *opts, "test", args.scope
    end
end

desc 'Run Stork Agent (debug mode)'
task :run_agent_debug, [:headless] => [DLV] + GO_AGENT_CODEBASE do |t, args|
    opts = []

    if args.headless == "true"
        opts = ["--headless", "-l", "0.0.0.0:45678"]
    end

    Dir.chdir("backend/cmd/stork-agent") do
        sh DLV, *opts, "debug"
    end
end

desc 'Run Stork Server (debug mode, no doc and UI)'
task :run_server_debug, [:headless, :ui_mode] => [DLV, :pre_run_server] + GO_SERVER_CODEBASE do |t, args|
    opts = []
    if args.headless == "true"
        opts = ["--headless", "-l", "0.0.0.0:45678"]
    end

    Dir.chdir("backend/cmd/stork-server") do
        sh DLV, *opts, "debug"
    end
end

################
### Frontend ###
################

desc 'Check frontend source code'
task :lint_ui => [NPX] + WEBUI_CODEBASE do
  Dir.chdir('webui') do
    sh NPX, "ng", "lint"
    sh NPX, "prettier", "--config", ".prettierrc", "--check", "**/*"
  end
end

desc 'Make frontend source code prettier'
task :fmt_ui => [NPX] + WEBUI_CODEBASE do
  Dir.chdir('webui') do
    sh NPX, "prettier", "--config", ".prettierrc", "--write", "**/*"
  end
end

# Globs of test files to include, relative to project root.
# There are 2 special cases:
#   when a path to directory is provided, all spec files ending ".spec.@(ts|tsx)" will be included
#   when a path to a file is provided, and a matching spec file exists it will be included instead
desc 'Run unit tests for UI.'
task :unittest_ui, [:test, :debug] => [NPX] + WEBUI_CODEBASE do |t, args|
    args.with_defaults(
        :debug => "false"
    )

    opts = []
    if args.test
        opts += ["--include", args.test]
    end

    opts += ["--progress", args.debug]
    opts += ["--watch", args.debug]

    opts += ["--browsers"]
    if args.debug == "true"
        opts += ["Chrome"]
    else
        opts += ["ChromeNoSandboxHeadless"]
    end

    Dir.chdir('webui') do
        sh NPX, "ng", "test", *opts
    end
end

############################
### Frontend - Debugging ###
############################

desc 'Build Stork UI (testing mode)'
task :build_ui_debug => [WEBUI_DEBUG_DIRECTORY]

desc 'Build Stork UI (testing mode) continuously whenever source files change'
task :build_ui_live => [NPX] + WEBUI_CODEBASE do
    Dir.chdir('webui') do
        sh NPX, "ng", "build", "--watch"
    end
end

#####################
### Documentation ###
#####################

desc 'Builds Stork documentation continuously whenever source files change'
task :build_doc_live => DOC_CODEBASE do
    Open3.pipeline(
        ['printf', '%s\\n', *DOC_CODEBASE],
        ['entr', '-d', 'rake', 'build_doc']
    )
end

#################
### Simulator ###
#################

flask_requirements_file = "tests/sim/requirements.txt"
flask = File.expand_path("tools/python/bin/flask")
file flask => [flask_requirements_file] do
    sh "pip3", "install",
            *ci_opts,
            "--force-reinstall",
            "--upgrade",
            "--no-input",
            "--no-deps",
            "--target", ENV["PYTHONPATH"],
            "-r", flask_requirements_file
end

task :run_sim => [flask] do
    ENV["STORK_SERVER_URL"] = "http://localhost:8080"
    ENV["FLASK_ENV"] = "development"
    ENV["FLASK_APP"] = "sim.py"
    ENV["LC_ALL"]  = "C.UTF-8"
    ENV["LANG"] = "C.UTF-8"

    Dir.chdir('tests/sim') do
        sh flask, "run", "--host", "0.0.0.0", "--port", "5005"
    end
end

#############
### Other ###
#############

task :lint_git => [DANGER] do
    if ENV["CI"] == nil
        puts "Warning! You cannot run this command locally."
    end
    sh DANGER, "--fail-on-errors=true", "--new-comment"
end

desc 'Migrate (and create) database to the newest version'
task :db_migrate, [:dbname, :dbhost, :dbport, :dbuser, :dbpass, :dbtrace] => [TOOL_BINARY_FILE] do |t, args|
    args.with_defaults(
        :dbname => ENV["POSTGRES_DB"] || "storktest",
        :dbhost => ENV["POSTGRES_ADDR"] || "localhost",
        :dbport => "5432",
        :dbuser => ENV["POSTGRES_USER"] || "storktest",
        :dbpass => ENV["POSTGRES_PASSWORD"] || "storktest",
        :dbtrace => "false"
    )

    dbhost = args.dbhost
    dbport = args.dbport
    if dbhost.include? ':'
        dbhost, dbport = dbhost.split(':')
    end
    
    ENV["STORK_DATABASE_HOST"] = dbhost
    ENV["STORK_DATABASE_PORT"] = dbport
    ENV["STORK_DATABASE_USER_NAME"] = args.dbuser
    ENV["STORK_DATABASE_PASSWORD"] = args.dbpass
    ENV["STORK_DATABASE_NAME"] = args.dbname
    ENV["STORK_DATABASE_TRACE"] = args.dbtrace
    
    ENV['PGPASSWORD'] = args.dbpass
    
    # Ignore error if DB already exist
    system "createdb",
        "-h", dbhost,
        "-p", dbport,
        "-U", args.dbuser,
        "-O", args.dbuser,
        args.dbname

    sh TOOL_BINARY_FILE, "db-up",
        "-d", args.dbname,
        "-u", args.dbuser,
        "--db-host", dbhost,
        "-p", dbport
end

desc "Remove remaing_test_databases_and_users"
task :db_remove_remaining, [:dbname, :dbhost, :dbport, :dbuser, :dbpass] do |t, args|
    args.with_defaults(
        :dbname => ENV["POSTGRES_DB"] || "storktest",
        :dbhost => ENV["POSTGRES_ADDR"] || "localhost",
        :dbport => "5432",
        :dbuser => ENV["POSTGRES_USER"] || "storktest",
        :dbpass => ENV["POSTGRES_PASSWORD"] || "storktest",
        :dbtrace => "false"
    )

    dbhost = args.dbhost
    dbport = args.dbport
    if dbhost.include? ':'
        dbhost, dbport = dbhost.split(':')
    end

    ENV['PGPASSWORD'] = args.dbpass

    psql_access_opts = [
        "-h", dbhost,
        "-p", dbport,
        "-U", args.dbuser
    ]

    psql_select_opts = [
        "-t",
        "-q",
        "-X",
    ]

    Open3.pipeline([
        "psql", *psql_select_opts, *psql_access_opts,
        "-c", "SELECT datname FROM pg_database WHERE datname ~ '#{args.dbname}.+'"
    ], [
        "xargs", "-P", "16", "-n", "1", "dropdb", *psql_access_opts 
    ])

    Open3.pipeline([
        "psql", *psql_select_opts, *psql_access_opts,
        "-c", "SELECT usename FROM pg_user WHERE usename ~ '#{args.dbname}.+'"
    ], [
        "xargs", "-P", "16", "-n", "1", "dropuser", *psql_access_opts 
    ])
end
