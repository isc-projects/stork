# Utilities
# The file contains helpful tasks
# for developers that aren't strictly
# related to the source code. 

namespace :utils do
    desc 'Generate ctags for Emacs'
    task :ctags => [ETAGS_CTAGS] do
        sh ETAGS_CTAGS,
        "-f", "TAGS",
        "-R",
        "--exclude=webui/node_modules",
        "--exclude=webui/dist",
        "--exclude=tools",
        "."
    end
    
    
    desc 'Connect gdlv GUI Go debugger to waiting dlv debugger'
    task :connect_dbg => [GDLV] do
        sh GDLV, "connect", "127.0.0.1:45678"
    end

    desc "List dependencies of a given package
        Choose one:
            ABS - full absolute package import path
            REL - package path relative to main Stork directory"
    task :list_package_deps => [GO] do
        Dir.chdir "backend" do
            if !ENV["ABS"].nil?
                package = ENV["ABS"]
            elsif !ENV["REL"].nil?
                if ENV["REL"].start_with? "backend/"
                    ENV["REL"] = ENV["REL"].delete_prefix "backend/"
                end
                
                package = File.join("isc.org/stork", ENV["REL"])
            else
                fail "You need to provide the ABS or REL variable"
            end
     
            stdout, _ = Open3.capture2 GO, "list", "-f", '# Package - import: {{ .ImportPath }} name: {{ .Name }}', package
            puts stdout

            stdout, _ = Open3.capture2 GO, "list", "-f", '{{ join .Deps "\n" }}', package

            std_deps = []
            external_deps = []

            stdout.split("\n").each do |d|
                stdout, _ = Open3.capture2 GO, "list", "-f", '{{ .Standard }}', d
                if stdout.strip == "true"
                    std_deps.append d
                else
                    external_deps.append d
                end
            end

            puts "# Dependency packages from standard library"
            std_deps.each do |d|
                puts d
            end

            puts
            puts "# External dependency packages"
            external_deps.each do |d|
                puts d
            end
        end
    end

    desc "List platforms supported by the installed Go version"
    task :list_go_supported_platforms => [GO] do
        sh GO, "tool", "dist", "list"
    end
end


namespace :prepare do
    desc 'Install the external dependencies related to the codebase'
    task :utils do
        find_and_prepare_deps(__FILE__)
    end
end


namespace :check do
    desc 'Check the external dependencies related to the utils'
    task :utils do
        check_deps(__FILE__)
    end
end
