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
        MODULE - the GO module name associated with the hook - required
        HOOK_DIR - the hook (plugin) directory - optional, default: #{DEFAULT_HOOK_DIRECTORY}"
    task :init => [GO] do
        module_name = ENV["MODULE"]
        if module_name.nil?
            fail "You must provide the MODULE variable with the module name"
        end

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY
        
        module_directory_name = module_name.gsub(/[^\w\.-]/, '_')

        destination = File.expand_path(File.join(hook_directory, module_directory_name))

        require 'pathname'
        main_module = "isc.org/stork@v0.0.0"
        main_module_directory_abs = Pathname.new('backend').realdirpath
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

    desc "Build all hooks. Remap plugins to use the current codebase.
        DEBUG - build plugins in debug mode - default: false
        HOOK_DIR - the hook (plugin) directory - optional, default: #{DEFAULT_HOOK_DIRECTORY}"
    task :build => [GO, :remap_core] do
        require 'tmpdir'

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

        # Removes old plugins
        puts "Removing old compiled hooks..."
        sh "rm", "-f", *FileList[File.join(hook_directory, "*.so")]

        mod_files = ["go.mod", "go.sum"]

        forEachHook(lambda { |dir_name|
            # Make a backup of the original mod files
            Dir.mktmpdir do |temp|
                sh "cp", *mod_files, temp

                puts "Building #{dir_name}..."
                sh "rake", "build"
                sh "cp", *FileList["build/*.so"], hook_directory

                # Back the changes in Go mod files.
                puts "Reverting remap operation..."
                sh "cp", *mod_files.collect { |f| File.join(temp, f) }, "."
            end
        })

        # The plugin filenames after remap lack the version.
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
        HOOK_DIR - the hook (plugin) directory - optional, default: #{DEFAULT_HOOK_DIRECTORY}
        COMMIT - use the given commit from the remote repository, if specified but empty use the current hash - optional
        TAG - use the given tag from the remote repository, if specified but empty use the current version as tag - optional
        If no COMMIT or TAG are specified then it remaps to use the local project."
    task :remap_core => [GO] do
        main_module = "isc.org/stork"
        main_module_directory_abs = File.expand_path "backend"
        remote_url = "gitlab.isc.org/isc-projects/stork/backend"
        core_commit, _ = Open3.capture2 "git", "rev-parse", "HEAD"

        forEachHook(lambda { |dir_name|
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
                require 'pathname'
                main_directory_abs_obj = Pathname.new(main_module_directory_abs)
                module_directory_abs_obj = Pathname.new(".").realdirpath
                module_directory_rel_obj = main_directory_abs_obj.relative_path_from module_directory_abs_obj

                target = module_directory_rel_obj.to_s
            end

            sh GO, "mod", "edit", "-replace", "#{main_module}=#{target}"
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
        HOOK_DIR - the hook (plugin) directory - optional, default: #{DEFAULT_HOOK_DIRECTORY}"
    task :server_hooks => ["hook:build"] do
        hook_directory = ENV["HOOK_DIR"] || ENV["STORK_SERVER_HOOK_DIRECTORY"] || DEFAULT_HOOK_DIRECTORY
        ENV["STORK_SERVER_HOOK_DIRECTORY"] = hook_directory
        Rake::Task["run:server"].invoke()
    end
end
