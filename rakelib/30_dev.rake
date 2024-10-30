# Development
# This file defines development-stage tasks,
# e.g., unit testing, linting, or debugging.

###############
### Files ###
###############

go_codebase = GO_SERVER_CODEBASE +
              GO_AGENT_CODEBASE +
              GO_TOOL_CODEBASE

# The temporary file generated by Storybook.
CLEAN.append "webui/documentation.json"

python_requirement_files = [
    "doc/src/requirements.in",
    "tests/sim/requirements.in",
] + FileList["rakelib/init_deps/*.in"]

#################
### Functions ###
#################

# Creates a temporary directory and removes it after the block execution.
# Source: https://stackoverflow.com/a/8791484
def in_tmpdir
    require 'tmpdir'
    path = File.expand_path "#{Dir.tmpdir}/#{Time.now.to_i}#{rand(1000)}/"
    FileUtils.mkdir_p path
    yield path
ensure
    FileUtils.rm_rf(path) if File.exists?(path)
end

# Open a file using the default application.
def open_file(path)
    if OS == "macos"
        program = "open"
    elsif OS == "linux" || OS == "FreeBSD"
        program = "xdg-open"
    else
        fail "operating system (#{OS}) not supported"
    end

    system program, path
end

#############
### Tasks ###
#############

namespace :fmt do
    desc 'Make frontend source code prettier.
        SCOPE - the files that the prettier should process, relative to webui directory - default: **/*'
    task :ui => [NPX] + WEBUI_CODEBASE do
        scope = "**/*"
        if !ENV["SCOPE"].nil?
            scope = ENV["SCOPE"]
        end
        Dir.chdir('webui') do
            sh NPX, "prettier", "--config", ".prettierrc", "--write", scope
        end
    end

    desc 'Format backend source code.
        SCOPE - the files that should be formatted, relative to the backend directory - default: ./...'
    task :backend => [GO] + go_codebase do
        scope = "./..."
        if !ENV["SCOPE"].nil?
            scope = ENV["SCOPE"]
        end
        Dir.chdir('backend') do
            sh GO, "fmt", scope
        end
    end

    desc 'Format Python source code'
    task :python => [BLACK] do
        python_files, exit_code = Open3.capture2('git', 'ls-files', '*.py')
        python_files = python_files.split("\n").map{ |string| string.strip }

        sh BLACK, *python_files
    end
end


namespace :unittest do
    desc 'Run unit tests for UI.
        TEST - globs of test files to include, relative to root or webui directory - default: unspecified
            There are 2 special cases:
                when a path to directory is provided, all spec files ending ".spec.@(ts|tsx)" will be included
                when a path to a file is provided, and a matching spec file exists it will be included instead
        DEBUG - run the tests in debug mode (no headless) - default: false'
    task :ui => [CHROME, NPX] + WEBUI_CODEBASE do
        debug = "false"
        if ENV["DEBUG"] == "true"
            debug = "true"
        end

        opts = []
        if !ENV["TEST"].nil?
            # The IDE built-in feature to copy a relative path returns the value
            # that starts from the repository's root. But this option requires
            # the path relative to the main frontend directory. The below line
            # allows us to provide both a path relative to the root or webui
            # directory.
            test_path = ENV["TEST"].delete_prefix('webui/')
            opts += ["--include", test_path]
        end

        opts += ["--progress", debug]
        opts += ["--watch", debug]

        opts += ["--browsers"]
        if debug == "true"
            opts += ["Chrome"]
        else
            opts += ["ChromeNoSandboxHeadless"]
        end

        Dir.chdir('webui') do
            sh NPX, "ng", "test", *opts
        end
    end

    desc 'Run backend unit and coverage tests
        SCOPE - Scope of the tests - default: all files
        TEST - Test name pattern to run - default: empty
        BENCHMARK - Execute benchmarks - default: false
        SHORT - Run short test routine - default: false
        HEADLESS - Run in headless mode - default: false
        VERBOSE - Print results for successful cases - default: false
        See "db:migrate" task for the database-related parameters
    '
    task :backend => [GO, TPARSE, GO_JUNIT_REPORT, "db:remove_remaining", "db:migrate", "gen:backend:mocks"] + go_codebase do
        scope = ENV["SCOPE"] || "./..."
        benchmark = ENV["BENCHMARK"] || "false"
        short = ENV["SHORT"] || "false"
        verbose = ENV["VERBOSE"] || "false"

        opts = []

        if !ENV["TEST"].nil?
            opts += ["-run", ENV["TEST"]]
        end

        if benchmark == "true"
            opts += ["-bench=."]
        end

        if short == "true"
            opts += ["-short"]
        end

        if verbose == "true"
            opts += ["-v"]
        end

        with_cov_tests = scope == "./..." && ENV["TEST"].nil?

        if with_cov_tests
            opts += ["-coverprofile=coverage.out"]

            at_exit {
                sh "rm -f backend/coverage.out"
            }
        end

        tparse_otps = []
        if ENV["CI"] == "true"
            tparse_otps.append("-nocolor")
        end

        Dir.chdir('backend') do
            statuses = Open3.pipeline(
                [GO, "test", "-json", *opts, "-race", scope],
                [GO_JUNIT_REPORT, "-iocopy", "-out", "./junit.xml"],
                [TPARSE, "-progress", *tparse_otps]
            )
            status = statuses[0]

            if !status.success?
                fail "Unit tests failed with status #{status.exitstatus}"
            end

            if with_cov_tests
                out, _ = Open3.capture2 GO, "tool", "cover", "-func=coverage.out"

                problem = false
                out.each_line do |line|
                    if line.start_with? 'total:'
                        next
                    end

                    items = line.gsub(/\s+/m, ' ').strip.split(" ")
                    file = items[0]
                    func = items[1]
                    cov = items[2].strip()[0..-2].to_f
                    rel_path = file.gsub("isc.org/stork/", "backend/")

                    ignore_list = [
                        'DetectServices', 'RestartKea', 'Serve', 'BeforeQuery', 'AfterQuery',
                        'Identity', 'LogoutHandler', 'NewDatabaseSettings', 'ConnectionParams',
                        'Password', 'loggingMiddleware', 'GlobalMiddleware', 'Authorizer',
                        'Listen', 'Shutdown', 'SetupLogging', 'UTCNow', 'detectApps',
                        'prepareTLS', 'handleRequest', 'pullerLoop', 'Collect',
                        'collectTime', 'collectResolverStat', 'collectResolverLabelStat',

                        # The Output method of the "systemCommandExecutor" structure encapsulates the
                        # "exec.Command" call to allow mocking of the system response in unit tests. The
                        # "exec.Command" cannot be directly mocked, so it is impossible to test the "Output"
                        # method.
                        'Output',

                        # We spent a lot of time to try test the main agent function. It is a problematic
                        # function because it starts listening and blocks itself until receiving SIGINT.
                        # Unfortunately, the signal handler isn't registered immediately after the function
                        # begins but after a short period.
                        # The unit tests for it were very unstable and time-depends. Additionally, the value
                        # of these tests was relatively poor. This function shouldn't be executed by the unit
                        # tests but rather by system tests.
                        'runAgent',

                        # this requires interacting with terminal
                        'GetSecretInTerminal', 'IsRunningInTerminal', 'prompt',
                        'promptForMissingArguments',

                        # This file contains the wrapper for the "gopsutil" package to allow mocking
                        # of its calls. Due to the nature of the package, it is impossible to test the wrapper.
                        'isc.org/stork/agent/process.go',

                        # The main server function is currently untestable.
                        'isc.org/stork/cmd/stork-server/main.go',

                        # Skip auto-generated files.
                        'isc.org/stork/server/gen',
                        'isc.org/stork/api/agent.pb.go',
                        'isc.org/stork/api/agent_grpc.pb.go',

                        # Skip test utils.
                        # Testing coverage should ignore testutil because we don't require writing
                        # tests for testing code. They can still be written but we shouldn't fail
                        # if they are not.
                        'isc.org/stork/server/test',
                        'isc.org/stork/server/agentcomm/test',
                        'isc.org/stork/server/database/test',
                        'isc.org/stork/server/database/model/test',
                        'isc.org/stork/testutil',

                        # Skip hook boilerplate,
                        'isc.org/stork/hooksutil/boilerplate'
                    ]
                    if short == 'true'
                        ignore_list.concat(['setupRootKeyAndCert', 'setupServerKeyAndCert', 'SetupServerCerts',
                                            'ExportSecret'])
                    end

                    if cov < 35 and not ignore_list.include? func
                        # Check if the file the whole package is ignored.
                        should_ignore = false
                        ignore_list.each { |ignored|
                            if file.start_with? ignored
                                should_ignore = true
                                break
                            end
                        }
                        if not should_ignore
                            puts "FAIL: %-80s %5s%% < 35%%" % ["#{rel_path} #{func}", "#{cov}"]
                            problem = true
                        end
                    end
                end

                if problem
                    fail("\nFAIL: Tests coverage is too low, add some tests\n\n")
                end
            end
        end
    end

    namespace :backend do
        desc 'Run backend unit tests (debug mode)
            SCOPE - Scope of the tests - required
            HEADLESS - Run in headless mode - default: false
            See "db:migrate" task for the database-related parameters'
        task :debug => [DLV, "db:remove_remaining", "db:migrate", "gen:backend:mocks"] + go_codebase do
            if ENV["SCOPE"].nil?
                fail "Scope argument is required"
            end

            opts = []

            if ENV["HEADLESS"] == "true"
                opts = ["--headless", "-l", "0.0.0.0:45678"]
            end

            Dir.chdir('backend') do
                sh DLV, *opts, "test", ENV["SCOPE"]
            end
        end

        desc 'Show backend coverage of unit tests in web browser
            See "db:migrate" task for the database-related parameters'
        task :cov => [GO, "unittest:backend"] do
            if !ENV["SCOPE"].nil?
                fail "Environment variable SCOPE cannot be specified"
            end

            if !ENV["TEST"].nil?
                fail "Environment variable TEST cannot be specified"
            end

            puts "Warning: Coverage may not work under Chrome-like browsers; use Firefox if any problems occur."
            Dir.chdir('backend') do
                sh GO, "tool", "cover", "-html=coverage.out"
            end
        end

        desc 'Profile backend unit tests
            SCOPE - Scope (package) of the tests. It may omit the `isc.org/stork` prefix. - required
            TEST - Test name pattern to run - default: empty
            KIND - Type of profile to generate - choice: cpu, mem, mutex, block; default: cpu
            See "db:migrate" task for the database-related parameters'
        task :profile => [GO, "db:remove_remaining", "db:migrate", "gen:backend:mocks"] + go_codebase do
            if ENV["TEST"].nil?
                puts "You must specify the test name pattern to run because "+
                    "profiling all tests at once is useless because most calls "+
                    "are made in the unit test framework itself."
                puts "Example: rake unittest:backend:profile TEST=^TestFoo$"
                fail "Environment variable TEST must be specified"
            end

            kind = ENV["KIND"] || "cpu"
            if !["cpu", "mem", "mutex", "block"].include? kind
                fail "Invalid profile kind: #{kind}, must be one of: cpu, mem, mutex, block"
            end

            if ENV["SCOPE"].nil?
                puts "Scope argument is required. It must be set to the "+
                    "package name of the test(s). It may omit the "+
                    "`isc.org/stork` prefix."
                puts "Example: rake unittest:backend:profile SCOPE=agent or "+
                    "rake unittest:backend:profile SCOPE=isc.org/stork/agent"
                fail "Environment variable SCOPE must be specified"
            end
            scope = ENV["SCOPE"]
            if !scope.start_with? "isc.org/stork/"
                scope = "isc.org/stork/#{scope}"
            end

            in_tmpdir do |tmpdir|
                profile_path = File.join tmpdir, "profile_#{kind}.prof"
                code_path = File.join tmpdir, "profile_#{kind}.test"

                # Generate profile file.
                Dir.chdir('backend') do
                    sh GO, "test",
                        "-run", ENV["TEST"], scope,
                        "--#{kind}profile", profile_path,
                        "-o", code_path,
                        "-count", "1"
                end

                sh GO, "tool", "pprof", "-http=:", code_path, profile_path
            end
        end
    end
end


namespace :build do
    desc 'Builds Stork documentation continuously whenever source files change'
    task :doc_live => [ENTR] + DOC_USER_CODEBASE + DOC_DEV_CODEBASE do
        Open3.pipeline(
            ['printf', '%s\\n', *DOC_USER_CODEBASE, *DOC_DEV_CODEBASE],
            [ENTR, '-d', 'rake', 'build:doc']
        )
    end

    desc 'Build Stork backend continuously whenever source files change'
    task :backend_live => go_codebase do
        Open3.pipeline(
            ['printf', '%s\\n', *go_codebase],
            [ENTR, '-d', 'rake', 'build:backend']
        )
    end

    desc 'Build Stork UI (testing mode)'
    task :ui_debug => [WEBUI_DEBUG_DIRECTORY]


    desc 'Build Stork UI (testing mode) continuously whenever source files change'
    task :ui_live => [NPX] + WEBUI_CODEBASE do
        Dir.chdir('webui') do
            sh NPX, "ng", "build", "--watch"
        end
    end
end


namespace :run do
    desc 'Run simulator'
    task :sim => [FLASK, FLAMETHROWER, DIG, PERFDHCP] do
        ENV["STORK_SERVER_URL"] = "http://localhost:8080"
        ENV["FLASK_ENV"] = "development"
        ENV["FLASK_APP"] = "sim.py"
        ENV["LC_ALL"]  = "C.UTF-8"
        ENV["LANG"] = "C.UTF-8"

        Dir.chdir('tests/sim') do
            sh FLASK, "run", "--host", "0.0.0.0", "--port", "5005"
        end
    end

    desc "Run Stork Server (debug mode, no doc and UI)
        HEADLESS - run debugger in headless mode - default: false
        UI_MODE - WebUI mode to use, must be build separately - choose: 'production', 'testing', 'none' or unspecify
        DB_TRACE - trace SQL queries - default: false"
    task :server_debug => [DLV, "db:setup_envvars", :pre_run_server] + GO_SERVER_CODEBASE do
        opts = []
        debug_opts = []
        if ENV["HEADLESS"] == "true"
            opts = ["--headless", "-l", "0.0.0.0:45678"]
            debug_opts.append "--continue"
        end

        Dir.chdir("backend/cmd/stork-server") do
            sh DLV, *opts, "debug",
                "--accept-multiclient",
                "--log",
                "--api-version", "2",
                *debug_opts
        end
    end

    desc 'Run Stork Agent (debug mode)
        PORT - agent port to use - default: 8888
        HEADLESS - run debugger in headless mode - default: false'
    task :agent_debug => [DLV] + GO_AGENT_CODEBASE do
        opts = []
        app_opts = []

        if ENV["HEADLESS"] == "true"
            opts = ["--headless", "-l", "0.0.0.0:45678"]
        end

        if ENV["PORT"].nil?
            ENV["PORT"] = "8888"
        end

        app_opts.append "--port", ENV["PORT"]

        Dir.chdir("backend/cmd/stork-agent") do
            sh DLV, *opts, "debug", "--", *app_opts
        end
    end

    desc 'Open the documentation in the browser'
    task :doc => [DOC_USER_ROOT, DOC_DEV_ROOT] do
        open_file "#{DOC_USER_ROOT}/index.html"
        open_file "#{DOC_DEV_ROOT}/index.html"
    end
end


namespace :lint do
    desc "Run danger commit linter"
    task :git => [DANGER] do
        if ENV["CI"] != "true"
            puts "Warning! You cannot run this command locally."
        end
        sh DANGER, "--fail-on-errors=true", "--new-comment"
    end

    desc 'Check frontend source code
        FIX - fix linting issues - default: false'
    task :ui => [NPX] + WEBUI_CODEBASE do
        ng_opts = []
        prettier_opts = []
        if ENV["FIX"] == "true"
            ng_opts += ["--fix"]
            prettier_opts += ["--write"]
        end

        Dir.chdir('webui') do
            sh NPX, "ng", "lint", *ng_opts
            sh NPX, "prettier", "--config", ".prettierrc", "--check", "**/*", *prettier_opts
        end
    end

    desc 'Check backend source code
        FIX - fix linting issues - default: false'
    task :backend => [GOLANGCILINT, "gen:backend:mocks"] + go_codebase do
        opts = []
        if ENV["FIX"] == "true"
            opts += ["--fix"]
        end

        Dir.chdir("backend") do
            sh GOLANGCILINT, "run", *opts
        end
    end

    desc 'Check shell scripts
        FIX - fix linting issues - default: false'
    task :shell => [GIT, SHELLCHECK] do
        # Get all files committed to git that have shell-specific terminations.
        files = []
        Open3.pipeline_r(
            [GIT, "ls-files"],
            ["grep", "-E", "\.sh$|\.prerm$|\.postinst"],
        ) {|output|
          output.each_line {|line|
            files.append line.rstrip
          }
        }

        # Add other files that are missing terminatons or ar more difficult to match.
        files.append 'utils/git-hooks/prepare-commit-msg'
        files.append 'utils/git-hooks-install'

        # Do the checking or fixing.
        if ENV["FIX"] == "true"
            Open3.pipeline(
                [SHELLCHECK, "-f", "diff", *files],
                [GIT, "apply", "--allow-empty"],
            )
        else
            sh SHELLCHECK, *files
        end
    end

    desc 'Runs pylint and flake8, python linter tools'
    task :python => ['lint:python:pylint', 'lint:python:flake8', 'lint:python:black']

    namespace :python do
        desc 'Runs pylint, python linter tool'
        task :pylint => [PYLINT, PYTEST, FLASK, OPEN_API_GENERATOR_PYTHON_DIR, *GRPC_PYTHON_API_FILES] do
            python_files, exit_code = Open3.capture2('git', 'ls-files', '*.py')
            python_files = python_files.split("\n").map{ |string| string.strip }
            puts "Running pylint:"
            sh PYLINT, '--rcfile', '.pylint', *python_files
        end

        desc 'Runs flake8, python linter tool'
        task :flake8 => [FLAKE8] do
            python_files, exit_code = Open3.capture2('git', 'ls-files', '*.py')
            python_files = python_files.split("\n").map{ |string| string.strip }
            puts "Running flake8:"
            sh FLAKE8, '--config', '.flake8', '--color=auto', *python_files
        end

        desc 'Runs black, python linter tool
        To run it in fixing mode, please use fmt:python task'
        task :black => [BLACK] do
            python_files, exit_code = Open3.capture2('git', 'ls-files', '*.py')
            python_files = python_files.split("\n").map{ |string| string.strip }
            puts "Running black (check mode):"
            sh BLACK, "--check", *python_files
        end
    end

    desc 'Check unreleased changelog files
        FIX - fix linting issues - default: false'
    task :changelog => [FOLD, SED] do
        changelog_dir = 'changelog_unreleased'
        files = Dir.entries(changelog_dir)
        exit_code = 0

        files.each do |filename|

            # Skip hidden files.
            if filename.start_with? '.'
                next
            end

            file = File.join(changelog_dir, filename)

            # Filter out non-files.
            if !File.file?(file)
                next
            end

            lines_too_long = []
            File.readlines(file).each_with_index do |line, line_number|
                # Wrap rows to width 73 == 72 + newline. Historically, number 72 has something to do with punch cards.
                if line.length > 73
                    lines_too_long.append [line_number, line]
                    exit_code = 1
                end
            end
            if ! lines_too_long.empty?
                puts 'ERROR: Changelog entry ' + filename + ' has too long lines. Should be < 73 characters.'
                lines_too_long.each do |pair|
                    puts "    #{pair[0]} #{pair[1]} ( #{pair[1].length} characters )"
                end
            end
            if ! lines_too_long.empty? && ENV["FIX"] == "true"
                output = ''
                Open3.pipeline_rw [
                    # Wrap rows to width 73 == 72 + newline. Historically, number 72 has something to do with punch cards.
                    FOLD, '-sw', '73', file
                ], [
                    # Remove trailing blank spaces.
                    SED, 's/ *$//g'
                ] do |stdin, stdout, _ts|
                  output = stdout.read
                end
                File.open(file, 'w') do |file_w|
                    file_w.write(output)
                end
            end
        end
        exit exit_code
    end
end

namespace :profile do
    # Internal task to connect the profiler to a Go binary.
    task :go_app, [:host, :port] => [GO] do |t, args|
        # Profile
        profile_raw = ENV["PROFILE"] || "cpu"
        profile = profile_raw
        if profile == 'cpu'
            profile = 'profile'
        end

        # Duration
        duration = ENV["DURATION"]
        support_duration = ['allocs', 'block', 'mutex', 'profile']
        if !duration.nil? && !support_duration.include?(profile)
            fail "Duration is not supported for #{profile_raw} profile"
        end

        if support_duration.include? profile
            expected_duration = duration || 30
            puts "Please wait, it will take #{expected_duration} seconds..."
        end

        # Build URL
        queryParams = []
        if !duration.nil?
            queryParams.append "seconds=#{duration}"
        end

        url = "http://#{args.host}:#{args.port}/debug/pprof/#{profile}?#{queryParams.join('&')}"

        # Profiler options
        opts = []
        if !ENV["COMPARE"].nil?
            opts.append "-diff_base", ENV["COMPARE"]
        end
        if !ENV["SUBSTACT"].nil?
            opts.append "-base", ENV["SUBSTACT"]
        end

        puts "Profiling #{profile_raw} on #{args.host}:#{args.port}..."
        sh GO, "tool", "pprof", *opts, "-http=:", url
    end

    # Internal task to connect the live profiler to a Go binary.
    task :go_app_live, [:host, :port] => [GOLIVEPPROF] do |t, args|
        # Build URL
        url = "http://#{args.host}:#{args.port}/debug/pprof"

        puts "Profiling live on #{args.host}:#{args.port}..."
        sh GOLIVEPPROF, url
    end

    desc 'Run profiling on running Stork agent
        PROFILE - profile type - choice: allocs, block, goroutine, heap, mutex, threadcreate, cpu - default: cpu
        DURATION - duration in seconds - default: 30
        COMPARE - Path to base profile for comparison - optional
        SUBSTACT - Path to base profile for substraction - optional'
    task :agent => [GO] do
        Rake::Task["profile:go_app"].invoke("localhost", "6061")
    end

    desc 'Run live profiling on running Stork agent'
    task :agent_live => [GO] do
        Rake::Task["profile:go_app_live"].invoke("localhost", "6061")
    end

    desc 'Run profiling on running Stork server
        PROFILE - profile type - choice: allocs, block, goroutine, heap, mutex, threadcreate, cpu - default: cpu
        DURATION - duration in seconds - default: 30
        COMPARE - Path to base profile for comparison - optional
        SUBSTACT - Path to base profile for substraction - optional'
    task :server => [GO] do
        Rake::Task["profile:go_app"].invoke("localhost", "6060")
    end

    desc 'Run live profiling on running Stork server'
    task :server_live => [GO] do
        Rake::Task["profile:go_app_live"].invoke("localhost", "6060")
    end
end

namespace :audit do
    desc 'Check the UI security issues.
        FIX - fix the detected vulnerabilities - default: false
        FORCE - allow for breaking changes - default: false'
    task :ui => [NPM] do
        opts = []
        if ENV["FIX"] == "true"
            opts.append "fix"
            if ENV["FORCE"] == "true"
                opts.append "--force"
            end
        end

        Dir.chdir("webui") do
            sh NPM, "audit", *opts
        end
    end

    desc 'Check the backend security issues'
    task :backend => [GOVULNCHECK] + go_codebase do
        Dir.chdir("backend") do
            sh GOVULNCHECK, "./..."
        end
    end

    desc 'Check the backend security issues (including testing codebase)'
    task :backend_tests => [GOVULNCHECK, "gen:backend:mocks"] + go_codebase do
        Dir.chdir("backend") do
            sh GOVULNCHECK, "-test", "./..."
        end
    end

    desc 'Check the Python security issues'
    task :python => [PIP_AUDIT] do
        opts = []
        python_requirement_files.each do |r|
            opts.append "-r", r.ext('txt')
        end

        sh PIP_AUDIT, *opts
    end
end


namespace :db do
    desc 'Setup the database environment variables
        DB_NAME - database name - default: env:POSTGRES_DB or storktest
        DB_HOST - database host - default: env:POSTGRES_ADDR or empty
        DB_PORT - database port - default: 5432
        DB_USER - database user - default: env:POSTGRES_USER or storktest
        DB_PASSWORD - database password - default: env: POSTGRES_PASSWORD or storktest
        DB_TRACE - trace SQL queries - default: false
        DB_MAINTENANCE_NAME - maintanance database name - default: postgres
        DB_MAINTENANCE_USER - maintannce username - default: postgres
        DB_MAINTENANCE_PASSWORD - maintenance password - default: empty'
    task :setup_envvars do
        dbname = ENV["STORK_DATABASE_NAME"] || ENV["DB_NAME"] || ENV["POSTGRES_DB"] || "storktest"
        dbhost = ENV["STORK_DATABASE_HOST"] || ENV["DB_HOST"] || ENV["POSTGRES_ADDR"] || ""
        dbport = ENV["STORK_DATABASE_PORT"] || ENV["DB_PORT"] || "5432"
        dbuser = ENV["STORK_DATABASE_USER_NAME"] || ENV["DB_USER"] || ENV["POSTGRES_USER"] || "storktest"
        dbpass = ENV["STORK_DATABASE_PASSWORD"] || ENV["DB_PASSWORD"] || ENV["POSTGRES_PASSWORD"] || "storktest"
        dbtrace = ENV["DB_TRACE"] || "false"
        dbmaintenance = ENV["STORK_DATABASE_MAINTENANCE_NAME"] || ENV["DB_MAINTENANCE_NAME"] || "postgres"
        dbmaintenanceuser = ENV["STORK_DATABASE_MAINTENANCE_USER_NAME"] || ENV["DB_MAINTENANCE_USER"] || "postgres"
        dbmaintenancepassword = ENV["STORK_DATABASE_MAINTENANCE_PASSWORD"] || ENV["DB_MAINTENANCE_PASSWORD"]

        if dbhost.include? ':'
            dbhost, dbport = dbhost.split(':')
        end

        ENV["STORK_DATABASE_HOST"] = dbhost
        ENV["STORK_DATABASE_PORT"] = dbport
        ENV["STORK_DATABASE_USER_NAME"] = dbuser
        ENV["STORK_DATABASE_PASSWORD"] = dbpass
        ENV["STORK_DATABASE_NAME"] = dbname
        ENV["STORK_DATABASE_MAINTENANCE_NAME"] = dbmaintenance
        ENV["STORK_DATABASE_MAINTENANCE_USER_NAME"] = dbmaintenanceuser
        ENV["STORK_DATABASE_MAINTENANCE_PASSWORD"] = dbmaintenancepassword

        if ENV["STORK_DATABASE_TRACE"].nil? && dbtrace == "true"
            ENV["STORK_DATABASE_TRACE"] = "run"
        end

        ENV['PGPASSWORD'] = dbpass
    end

    desc 'Migrate (and create) database to the newest version
        FORCE_MIGRATION - reset database to the initial state and perform all migration again - default: false
        See db:setup_envvars task for more options.'
    task :migrate => [:setup_envvars, TOOL_BINARY_FILE] do
        sh TOOL_BINARY_FILE, "db-create"
        sh TOOL_BINARY_FILE, "db-init"
        if ENV["FORCE_MIGRATION"] == "true"
            sh TOOL_BINARY_FILE, "db-reset"
        end
        sh TOOL_BINARY_FILE, "db-up"
    end

    desc "Remove remaining test databases and users
        See db:setup_envvars task for more options."
    task :remove_remaining => [PSQL, DROPUSER, DROPDB, :setup_envvars] do
        dbhost = ENV["STORK_DATABASE_HOST"]
        dbuser = ENV["STORK_DATABASE_USER_NAME"]
        dbport = ENV["STORK_DATABASE_PORT"]
        dbname = ENV["STORK_DATABASE_NAME"]
        dbmaintenancename = ENV["STORK_DATABASE_MAINTENANCE_NAME"]
        dbmaintenanceuser = ENV["STORK_DATABASE_MAINTENANCE_USER_NAME"]
        dbmaintenancepass = ENV["STORK_DATABASE_MAINTENANCE_PASSWORD"]

        ENV["PGPASSWORD"] = dbmaintenancepass

        psql_access_opts = [
            "-h", dbhost,
            "-p", dbport,
            "-U", dbmaintenanceuser
        ]

        psql_select_opts = [
            "-t",
            "-q",
            "-X",
        ]

        # Don't destroy the pattern database
        dbname_pattern = "#{dbname}.+"

        Open3.pipeline([
            PSQL, *psql_select_opts, *psql_access_opts, dbmaintenancename,
            "-c", "SELECT datname FROM pg_database WHERE datname ~ '#{dbname_pattern}'"
        ], [
            # Remove empty rows
            "awk", "NF"
        ], [
            "xargs", "-P", "16", "-n", "1", "-r", DROPDB, *psql_access_opts
        ])

        Open3.pipeline([
            PSQL, *psql_select_opts, *psql_access_opts, dbmaintenancename,
            "-c", "SELECT usename FROM pg_user WHERE usename ~ '#{dbuser}.+'"
        ], [
            # Remove empty rows
            "awk", "NF"
        ], [
            "xargs", "-P", "16", "-n", "1", "-r", DROPUSER, *psql_access_opts
        ])
    end
end


desc 'Run Storybook
    CACHE - use internal Storybook cache, disable for fix the "Cannot GET /" problem - default: true'
task :storybook => [NPX] + WEBUI_CODEBASE do
    opts = []
    if ENV["CACHE"] == "false"
        opts.append "--no-manager-cache"
    end

    ENV["STORYBOOK_DISABLE_TELEMETRY"] = "1"

    Dir.chdir("webui") do
        sh NPX, "ng", "run", "stork:storybook", "--", *opts
    end
end


namespace :gen do
    namespace :ui do
        desc 'Generate Angular stuff. Pass through the arguments to
        "ng generate" command. They must be delimited by double dash (--).'
        task :angular => [NPX] do |t|
            flags = []
            found_delimiter = false

            ARGV.each do |arg|
                if arg == "--"
                    found_delimiter = true
                    next
                end

                next if !found_delimiter

                flags.append arg
            end

            if flags.empty?
                fail "No double dash (--) delimiter found."
            end

            Dir.chdir("webui") do
                sh NPX, "ng", "generate", *flags
            end
        end

        desc 'Generate Angular service
        NAME - name of the service - required'
        task :service => [NPX] do
            Dir.chdir("webui") do
                sh NPX, "ng", "generate", "service", ENV["NAME"]
            end
        end

        desc 'Regenerate package.json.lock'
        task :package_lock => [NPM] do
            Dir.chdir("webui") do
                sh NPM, "install", "--package-lock-only"
            end
        end
    end

    namespace :backend do
        desc 'Regenerate go.sum.'
        task :go_sum => [GO] do
            Dir.chdir("backend") do
                sh GO, "mod", "download", "-x"
            end
        end

        desc 'Generate all Go mocks'
        task :mocks => [GO, MOCKGEN, MOCKERY] + go_codebase do
            Dir.chdir("backend") do
                sh GO, "generate", "./..."
            end
        end
    end

    desc 'Regenerate Python requirements file'
    task :python_requirements => [PIP_COMPILE] do
        python_requirement_files.each do |r|
            sh PIP_COMPILE, "--strip-extras", r
        end
    end

    desc 'Regenerate Ruby lock file'
    task :ruby_gemlocks => [BUNDLE] do
        gemfiles = FileList["rakelib/init_deps/*/Gemfile"]
            .exclude(FileList["rakelib/init_deps/*/Gemfile.lock"])

        gemfiles.each do |g|
            gemfile_dir = File.dirname(g)
            Dir.chdir(gemfile_dir) do
                sh BUNDLE, "lock"
            end
        end
    end
end


namespace :update do
    desc 'Update Angular
    VERSION - target Angular version, hint: use only major and minor - required
    FORCE - ignore warnings - optional, default: false'
    task :angular => [NPX] do
        version=ENV["VERSION"]
        if version.nil?
            fail "Provide VERSION variable"
        end

        opts = []
        if ENV["FORCE"] == "true"
            opts.append "--force"
        end

        Dir.chdir("webui") do
            sh NPX, "ng", "update", *opts,
                "@angular/core@#{version}",
                "@angular/cli@#{version}",
                "@angular/cdk@#{version}"
        end
    end

    desc 'Update Angular ESLint
    VERSION - target ESLint version, hint: use only major and minor - required
    FORCE - ignore warnings - optional, default: false'
    task :angular_eslint => [NPX] do
        version=ENV["VERSION"]
        if version.nil?
            fail "Provide VERSION variable"
        end

        opts = []
        if ENV["FORCE"] == "true"
            opts.append "--force"
        end

        Dir.chdir("webui") do
            sh NPX, "ng", "update", *opts,
                "@angular-eslint/builder@#{version}",
                "@angular-eslint/eslint-plugin@#{version}",
                "@angular-eslint/eslint-plugin-template@#{version}",
                "@angular-eslint/schematics@#{version}",
                "@angular-eslint/template-parser@#{version}"
        end
    end

    desc 'Update Storybook to the latest version'
    task :storybook => [STORYBOOK] do
        Dir.chdir("webui") do
            sh STORYBOOK, "upgrade", "--disable-telemetry"
        end
    end

    desc 'Update internal browsers list. It makes changes in the package-lock file to fix the problems with out-of-date data.'
    task :browserslist => [NPX] do
        Dir.chdir("webui") do
            sh NPX, "browserslist", "--update-db"
        end
    end

    desc 'Update all npm dependencies to the "Wanted" versions (mainly updates to the latest minor).'
    task :ui_deps => [NPM] do
        Dir.chdir("webui") do
            sh NPM, "update"
            # Prints possible manual updates.
            sh NPM, "outdated"
        end
    end

    desc 'Update all go.mod dependencies to the latest versions.'
    task :backend_deps => [GO] do
        Dir.chdir("backend") do
            sh GO, "get", "-u", "-t", "./..."
            sh GO, "mod", "tidy"
        end
    end

    desc 'Update all Python dependencies
        DRY_RUN - do not update the packages, just re-generate using the local versions - default: false'
    task :python_requirements => [PIP_COMPILE, PIP_SYNC] do
        require 'pathname'

        opts = ["--strip-extras"]

        # Generate a layered requirements file that composes all requirement
        # files.
        all_requirements_dir = "rakelib/init_deps"
        all_requirements_dir_path = Pathname.new all_requirements_dir
        all_requirements_file_in = File.join(all_requirements_dir, "all.in")
        all_requirements_file_txt = all_requirements_file_in.ext('txt')

        File.open(all_requirements_file_in, 'w') do |file|
            python_requirement_files.each do |r|
                r_path = Pathname.new r
                r_rel_path = r_path.relative_path_from all_requirements_dir_path
                file.write("-c #{r_rel_path}\n")
            end
        end

        # Update all requirements at once to ensure there are no conflicts.
        update_opts = []
        if ENV["DRY_RUN"] != "true"
            update_opts.append "--upgrade"
        end

        sh PIP_COMPILE, *opts, *update_opts, "-o", all_requirements_file_txt, all_requirements_file_in
        # Install the updated versions.
        if ENV["DRY_RUN"] != "true"
            sh PIP_SYNC, all_requirements_file_txt
        end

        # Generate the separate requirements.txt files. It uses the previously
        # upgraded versions.
        python_requirement_files.each do |r|
            # The TXT files must be removed; otherwise, they will
            # not be updated due to missing the --upgrade flag.
            FileUtils.rm r.ext('txt')
            sh PIP_COMPILE, *opts, r
        end

        # Clean-up
        FileUtils.rm all_requirements_file_in
        FileUtils.rm all_requirements_file_txt
    end

    desc 'Update all Ruby dependencies'
    task :ruby_gemfiles => [BUNDLE] do
        gemfiles = FileList["rakelib/init_deps/*/Gemfile"]
            .exclude(FileList["rakelib/init_deps/*/Gemfile.lock"])
        # List all Gemfiles.
        gemfiles.each do |g|
            gemfile_dir = File.dirname(g)
            Dir.chdir(gemfile_dir) do
                # Update dependencies in the lock file.
                sh BUNDLE, "update"
            end
        end
    end
end


namespace :prepare do
    desc 'Install the external dependencies related to the development'
    task :dev do
        find_and_prepare_deps(__FILE__)
    end
end


namespace :check do
    desc 'Check the external dependencies related to the development'
    task :dev do
        check_deps(__FILE__)
    end
end
