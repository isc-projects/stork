# Hooks
# The file contains tasks to write and build the Stork hook libraries
# (GO plugins)

#############
### Files ###
#############

DEFAULT_HOOK_DIRECTORY = File.expand_path "plugins"

CLEAN.append *FileList[File.join(DEFAULT_HOOK_DIRECTORY, "*.so")]


#################
### Functions ###
#################

def forEachHook(f)
    hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

    Dir.foreach(hook_directory) do |dir_name|
        path = File.join(hook_directory, dir_name)
        next if dir_name == '.' or dir_name == '..' or !File.directory? path

        Dir.chdir(path) do
            f.call(dir_name)
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

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY
        
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

    desc 'Build all hooks. Remap plugins to use the current codebase.
        DEBUG - build plugins in debug mode - default: false
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins'
    task :build => [GO, :remap_core] do
        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

        # Removes old plugins
        puts "Removing old compiled hooks..."
        sh "rm", "-f", *FileList[File.join(hook_directory, "*.so")]

        forEachHook(lambda { |dir_name|
            puts "Building #{dir_name}..."
            sh "rake", "build"
            sh "cp", *FileList["build/*.so"], hook_directory

            # Back the changes in Go mod files.
            puts "Reverting remap operation..."
            sh "git", "checkout", "go.mod", "go.sum"
        })

        # The plugin filenames after remap doesn't contain the version.
        # We need to append it.
        commit, _ = Open3.capture2 "git", "rev-parse", "--short", "HEAD"
        commit = commit.strip()

        Dir[File.join(hook_directory, "*.so")].each do |path|
            new_path = File.join(
                File.dirname(path),
                "#{File.basename(path, ".so")}#{STORK_VERSION}-#{commit}.so"
            )
            sh "mv", path, new_path
        end
    end

    desc "Remap the dependency path to the Stork core. It specifies the source
        of the core dependency - remote repository or local directory. The
        remote repository may be fetched by tag or commit hash.
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins
        COMMIT - use the given commit from the remote repository, if specified but empty use the current hash - optional
        TAG - use the given tag from the remote repository, if specified but empty use the current version as tag - optional
        If no COMMIT or TAG are specified then it remaps to use the local project."
    task :remap_core => [GO] do
        main_package = "isc.org/stork"
        main_package_directory_abs = File.expand_path "backend"
        remote_url = "gitlab.isc.org/isc-projects/stork/backend"

        forEachHook(lambda { |dir_name|
            target = nil

            if !ENV["COMMIT"].nil?
                puts "Remap to use a specific commit"
                commit = ENV["COMMIT"]
                if commit == ""
                    commit, _ = Open3.capture2 "git", "rev-parse", "HEAD"
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
                require 'pathname'
                main_directory_abs_obj = Pathname.new(main_package_directory_abs)
                package_directory_abs_obj = Pathname.new(".").realdirpath
                package_directory_rel_obj = main_directory_abs_obj.relative_path_from package_directory_abs_obj

                target = package_directory_rel_obj.to_s
            end

            sh GO, "mod", "edit", "-replace", "#{main_package}=#{target}"
            sh GO, "mod", "tidy"
        })
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
    desc "Run Stork Server with hooks
        HOOK_DIR - the hook (plugin) directory - optional, default: plugins"
    task :server_hooks => ["hook:build"] do
        hook_directory = ENV["HOOK_DIR"] || ENV["STORK_SERVER_HOOK_DIRECTORY"] || DEFAULT_HOOK_DIRECTORY
        ENV["STORK_SERVER_HOOK_DIRECTORY"] = hook_directory
        Rake::Task["run:server"].invoke()
    end
end
