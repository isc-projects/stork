# Hooks
# The file contains tasks to write and build the Stork hook libraries
# (GO plugins)

#########################
### Constants & Files ###
#########################

MAIN_MODULE = "isc.org/stork"
MAIN_MODULE_DIRECTORY_ABS = File.expand_path "backend"

default_hook_directory_rel = "hooks"
DEFAULT_HOOK_DIRECTORY = File.expand_path default_hook_directory_rel

CLEAN.append *FileList[File.join(DEFAULT_HOOK_DIRECTORY, "*.so")]

pkg_directory = "dist/hook-pkgs"
directory pkg_directory
hook_nfpm_config_file = "etc/dist/hook.yaml"

CLEAN.append pkg_directory

#################
### Functions ###
#################

# Iterates over the hook directories and executes the given block for each
# of them.
# The block may accept three arguments:
#
# 1. Hook directory name
# 2. Absolute path to the hook directory
# 3. Absolute path to the subdirectory in the hook directory containing the
#    go.mod file (source subdirectory).
#
# The current working directory during the block execution is the subdirectory
# containing the go.mod file.
def forEachHook(&block)
    require 'find'

    hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

    Dir.foreach(hook_directory) do |dir_name|
        path = File.join(hook_directory, dir_name)
        next if dir_name == '.' or dir_name == '..' or !File.directory? path

        Dir.chdir(path) do
            project_path = File.expand_path path

            # Search for the go.mod
            src_path = nil

            Find.find '.' do |path|
                if File.basename(path) == 'go.mod'
                    src_path = File.dirname(path)
                    break
                end
            end

            if src_path.nil?
                fail 'Cannot find the go.mod file'
            end

            src_path = File.expand_path src_path

            Dir.chdir(src_path) do
                block.call(dir_name, project_path, src_path)
            end
        end
    end
end

# Remaps the go.mod file in the current working directory to use the local
# Stork core codebase.
def remap_core_local()
    require 'pathname'
    main_directory_abs_obj = Pathname.new(MAIN_MODULE_DIRECTORY_ABS)
    module_directory_abs_obj = Pathname.new(".").realdirpath
    module_directory_rel_obj = main_directory_abs_obj.relative_path_from module_directory_abs_obj

    target = module_directory_rel_obj.to_s

    sh GO, "mod", "edit", "-replace", "#{MAIN_MODULE}=#{target}"
    sh GO, "mod", "tidy"
end

#############
### Tasks ###
#############

namespace :hook do
    desc "Init new hook directory
        MODULE - the name  of the hook module used in the go.mod file and as the hook directory name - required
        HOOK_DIR - the directory containing the hooks - optional, default: #{default_hook_directory_rel}"
    task :init => [GO] do
        module_name = ENV["MODULE"]
        if module_name.nil?
            fail "You must provide the MODULE variable with the module name"
        end

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

        module_directory_name = module_name.gsub(/[^\w\.-]/, '_')

        destination = File.expand_path(File.join(hook_directory, module_directory_name))

        require 'pathname'
        main_module = "#{MAIN_MODULE}@v0.0.0"
        main_module_directory_abs = Pathname.new(MAIN_MODULE_DIRECTORY_ABS)
        module_directory_abs = Pathname.new(destination)
        module_directory_rel = main_module_directory_abs.relative_path_from module_directory_abs

        sh "mkdir", "-p", destination

        Dir.chdir(destination) do
            sh "git", "init"
            sh GO, "mod", "init", module_name
            sh GO, "mod", "edit", "-require", main_module
            sh GO, "mod", "edit", "-replace", "#{main_module}=#{module_directory_rel}"
            sh "touch", "go.sum"
        end

        sh "cp", *FileList["backend/hooksutil/boilerplate/*"], destination
    end

    desc "Build all hooks. Remap hooks to use the current codebase.
        DEBUG - build hooks in debug mode, the envvar is passed through to the hook Rakefile - default: false
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}"
    task :build => [GO] do
        require 'tmpdir'

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

        # Removes old hooks
        puts "Removing old compiled hooks..."
        sh "rm", "-f", *FileList[File.join(hook_directory, "*.so")]

        mod_files = ["go.mod", "go.sum"]

        forEachHook do |dir_name, project_path|
            # Make a backup of the original mod files
            Dir.mktmpdir do |temp|
                # Preserve the original mod files.
                sh "cp", *mod_files, temp

                # Remap the core dependency to the local directory.
                remap_core_local()

                # Compile the hook.
                puts "Building #{dir_name}..."
                sh "rake", "build"

                # Collect the compiled hook.
                compiled_binaries = FileList[File.join(project_path, "build/*.so")]
                if compiled_binaries.length == 0
                    fail "No compiled hook found in #{project_path}/build"
                end
                sh "cp", *compiled_binaries, hook_directory

                # Back the changes in Go mod files.
                puts "Reverting remap operation..."
                sh "cp", *mod_files.collect { |f| File.join(temp, f) }, "."
            end
        end
    end

    desc "Build all hooks and create packages. Remap hooks to use the current codebase.
        DEBUG - build hooks in debug mode, the envvar is passed through to the hook Rakefile - default: false
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}"
    task :build_pkg => [NFPM, TOOL_MAN_FILE, hook_nfpm_config_file, pkg_directory] do
        # Suppress (re)building the hook binaries if the SUPPRESS_PREREQUISITES
        # environment variable is set to true.
        if ENV["SUPPRESS_PREREQUISITES"] != "true"
            Rake::Task["hook:build"].invoke()
        end

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY
        pkg_type = get_package_manager_type()
        arch = get_target_go_arch()

        FileList[File.join(hook_directory, "*.so")].each do |hook_path|
            hook_filename = File.basename(hook_path)
            components = hook_filename.split("-", 3)
            if components.length != 3 || components[0] != "stork" || !hook_filename.end_with?(".so")
                fail "Invalid hook name: #{hook_filename}. It must follow the pattern: stork-<application>-<name>.so"
            end

            kind = components[1]
            hook_name = components[2].chomp(".so")

            man_directory = File.dirname(TOOL_MAN_FILE)
            man_extension = File.extname(TOOL_MAN_FILE)
            hook_filename_no_ext = File.basename(hook_filename, ".so")
            hook_man_path = File.join(man_directory, hook_filename_no_ext + man_extension)
            puts "Hook man path:", hook_man_path

            ENV["STORK_NFPM_ARCH"] = arch
            ENV["STORK_NFPM_VERSION"] = "#{STORK_VERSION}.#{TIMESTAMP}"
            ENV["STORK_NFPM_HOOK_KIND"] = kind
            ENV["STORK_NFPM_HOOK_NAME"] = hook_name
            ENV["STORK_NFPM_HOOK_PATH"] = hook_path
            ENV["STORK_NFPM_HOOK_FILENAME"] = hook_filename
            ENV["STORK_NFPM_HOOK_MAN_PATH"] = hook_man_path

            sh NFPM, "package",
                "--config", hook_nfpm_config_file,
                "--packager", pkg_type,
                "--target", pkg_directory
        end
    end

    desc "Lint hooks against the Stork core rules.
        FIX - fix linting issues - default: false
        HOOK_DIR - the directory containing the hooks - optional, default: #{default_hook_directory_rel}"
    task :lint => [GOLANGCILINT] do
        require 'pathname'

        opts = []
        if ENV["FIX"] == "true"
            opts += ["--fix"]
        end

        # Use relative path for more human-friendly linter output.
        hook_directory = Pathname.new(ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY)
        main_directory = Pathname.new Dir.pwd
        hook_directory_rel = hook_directory.relative_path_from main_directory
        config_path = File.expand_path "backend/.golangci.yml"

        forEachHook do |dir_name|
            sh GOLANGCILINT, "run",
                "-c",  config_path,
                "--path-prefix", File.join(hook_directory_rel, dir_name),
                *opts
        end
    end

    desc "Run the unit tests for all hooks."
    task :unittest => [GO] do
        forEachHook do |dir_name|
            # Check if the unit test task exists.
            stdout, _ = Open3.capture2 "rake", "-D", "^unittest$"
            if stdout.nil? || stdout.strip.empty?
                puts "Skipping #{dir_name} - no unit tests."
                next
            end

            # Run the unit tests.
            puts "Running unit tests for #{dir_name}..."
            sh "rake", "unittest"
        end
    end

    desc "Remap the dependency path to the Stork core. It specifies the source
        of the core dependency - remote repository or local directory. The
        remote repository may be fetched by tag or commit hash.
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}
        COMMIT - use the given commit from the remote repository, if specified but empty use the current hash - optional
        TAG - use the given tag from the remote repository, if specified but empty use the current version as tag - optional
        If no COMMIT or TAG are specified then it remaps to use the local project."
    task :remap_core => [GO] do
        remote_url = "gitlab.isc.org/isc-projects/stork/backend"
        core_commit, _ = Open3.capture2 "git", "rev-parse", "HEAD"

        forEachHook do |dir_name|
            target = nil

            if !ENV["COMMIT"].nil?
                puts "Remap to use a specific commit"
                commit = ENV["COMMIT"]
                if commit == ""
                    commit = core_commit
                end

                target = "#{remote_url}@#{commit}"
            elsif !ENV["TAG"].nil?
                puts "Remap to use a specific tag"
                tag = ENV["TAG"]
                if tag == ""
                    tag = STORK_VERSION
                end

                if !tag.start_with? "v"
                    tag = "v" + tag
                end

                target = "#{remote_url}@#{tag}"
            else
                puts "Remap to use the local directory"
                remap_core_local()
                next
            end

            sh GO, "mod", "edit", "-replace", "#{MAIN_MODULE}=#{target}"
            sh GO, "mod", "tidy"
        end
    end

    desc "List dependencies of a given callout specification package
        KIND - callout kind - required, choice: agent or server
        CALLOUT - callout specification (interface) package name - required"
    task :list_callout_deps => [GO] do
        kind = ENV["KIND"]
        if kind != "server" && kind != "agent"
            fail "You need to provide the callout kind in KIND variable: agent or server"
        end

        callout = ENV["CALLOUT"]
        if callout.nil?
            fail "You need to provide the callout package name in CALLOUT variable."
        end

        package_rel = "hooks/#{kind}/#{callout}"
        ENV["REL"] = package_rel
        Rake::Task["utils:list_package_deps"].invoke
    end

    desc "Fetches the hook repositories"
    task :prepare => [GIT] do
        # Initialize the hook submodules.
        sh GIT, "submodule", "update", "--init", "--recursive"
    end

    desc "Updates the submodule references to the latest commit in the hook
        repositories"
    task :sync => [GIT] do
        # Update the hook submodules.
        sh GIT, "submodule", "update", "--recursive"
    end

    desc "Prepare release tarball with Stork hook sources"
    task :tarball => [GIT] do
        root_abs = File.expand_path "."

        forEachHook do |dir_name, project_path|
            # The cwd in the forEachHook block is set to src/ subdirectory.
            Dir.chdir("..") do
                sh GIT, "archive",
                    "--prefix", "#{dir_name}-#{STORK_VERSION}/",
                    "-o", File.join(root_abs, "#{dir_name}-#{STORK_VERSION}.tar.gz"),
                    "HEAD"
            end
        end
    end
end

namespace :run do
    desc "Run Stork Server with hooks
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}"
    task :server_hooks => ["hook:build"] do
        hook_directory = ENV["HOOK_DIR"] || ENV["STORK_SERVER_HOOK_DIRECTORY"] || DEFAULT_HOOK_DIRECTORY
        ENV["STORK_SERVER_HOOK_DIRECTORY"] = hook_directory
        Rake::Task["run:server"].invoke()
    end
end
