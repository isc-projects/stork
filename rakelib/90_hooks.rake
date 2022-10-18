# Hooks
# The file contains tasks to write and build the Stork hook libraries
# (GO plugins)

#############
### Files ###
#############

CLEAN.append *FileList["plugins/*.so"]

#################
### Functions ###
#################

def forEachHook(f)
    hook_directory = "plugins"
    if !ENV["HOOK_DIR"].nil?
        hook_directory = ENV["HOOK_DIR"]
    end

    Dir.foreach(hook_directory) do |filename|
        path = File.join(hook_directory, filename)
        next if filename == '.' or filename == '..' or !File.directory? path

        Dir.chdir(path) do
            f.call(path)
        end
    end
end

#############
### Tasks ###
#############

namespace :hook do
    desc "Init new hook directory
        PACKAGE - the package name - required
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins"
    task :init => [GO] do
        package = ENV["PACKAGE"]
        if package.nil?
            fail "You must provide the PACKAGE variable with the package name"
        end

        hook_directory = "plugins"
        if !ENV["HOOK_DIR"].nil?
            hook_directory = ENV["HOOK_DIR"]
        end
        
        package_directory_name = package.gsub(/[^\w\.-]/, '_')

        destination = File.expand_path(File.join(hook_directory, package_directory_name))

        require 'pathname'
        main_package = "isc.org/stork@v0.0.0"
        main_package_directory_abs = Pathname.new('backend').realdirpath
        package_directory_abs = Pathname.new(destination)
        package_directory_rel = main_package_directory_abs.relative_path_from package_directory_abs

        sh "mkdir", "-p", destination
        Dir.chdir(destination) do
            sh "git", "init"
            sh GO, "mod", "init", package
            sh GO, "mod", "edit", "-require", main_package
            sh GO, "mod", "edit", "-replace", "#{main_package}=#{package_directory_rel}"
        end

        sh "cp", *FileList["backend/hooksutil/templates/*"], destination
    end

    desc "Build the hooks
        DEBUG - build plugins in debug mode - default: false
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins"
    task :build => [GO] do
        flags = []
        if ENV["DEBUG"] == "true"
            flags.append "-gcflags", "all=-N -l"
        end

        forEachHook(lambda { |path|
            filename = File.basename(path)
            sh GO, "mod", "tidy"
            sh GO, "build", *flags, "-buildmode=plugin", "-o", File.join("..", filename + ".so")
        })
    end

    desc "Lint the hooks
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins
        FIX - fix linting issues - default: false"
    task :lint => [GO] do
        config_path = File.expand_path "backend/.golangci.yml"

        opts = ["-c", config_path]
        if ENV["FIX"] == "true"
            opts += ["--fix"]
        end

        forEachHook(lambda { |path|
            sh GOLANGCILINT, "run", *opts 
        })
    end

    desc "Run hooks unit tests
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins"
    task :unittest => [RICHGO] do
        forEachHook(lambda { |path|
            sh RICHGO, "test", "-race", "-v", "./..." 
        })
    end


    desc "Fix relative paths to the Stork core. It should be used if the hook
    directory was moved or if the external plugin was fetched.
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins"
    task :fix_core_rel => [GO] do
        hook_directory = "plugins"
        if !ENV["HOOK_DIR"].nil?
            hook_directory = ENV["HOOK_DIR"]
        end

        Dir.foreach(hook_directory) do |filename|
            path = File.join(hook_directory, filename)
            next if filename == '.' or filename == '..' or !File.directory? path

            require 'pathname'
            main_package = "isc.org/stork@v0.0.0"
            main_package_directory_abs = Pathname.new('backend').realdirpath
            package_directory_abs = Pathname.new(path).realdirpath
            package_directory_rel = main_package_directory_abs.relative_path_from package_directory_abs

            Dir.chdir(path) do
                sh GO, "mod", "edit", "-replace", "#{main_package}=#{package_directory_rel}"
            end
        end
    end

    desc "List dependencies of a given callout package
        KIND - callout kind - required, choice: agent or server
        CALLOUT - callout package name - required"
    task :list_callout_deps => [GO] do
        kind = ENV["KIND"]
        if kind != "server" && kind != "agent"
            fail "You need to provide the callout kind in KIND variable: agent or server"
        end

        callout = ENV["CALLOUT"]
        if callout.nil?
            fail "You need to provide the callout name in CALLOUT variable."
        end

        package_rel = "hooks/#{kind}/#{callout}"
        ENV["REL"] = package_rel
        Rake::Task["utils:list_package_deps"].invoke
    end
end

namespace :run do
    desc "Run Stork Server with hooks"
    task :server_hooks => ["hook:build"] do
        ENV["STORK_SERVER_HOOK_DIRECTORY"] = File.expand_path "plugins"
        Rake::Task["run:server"].invoke()
    end
end
