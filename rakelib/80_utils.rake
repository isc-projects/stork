# Utilities
# The file contains helpful tasks
# for developers that aren't strictly
# related to the source code. 

namespace :utils do
    desc 'Generate ctags for Emacs'
    task :ctags do
        sh "etags.ctags",
        "-f", "TAGS",
        "-R",
        "--exclude=webui/node_modules",
        "--exclude=webui/dist",
        "--exclude=tools",
        "."
    end
    
    
    desc 'Connect gdlv GUI Go debugger to waiting dlv debugger'
    task :connect_dbg => GDLV do
        sh GDLV, "connect", "127.0.0.1:45678"
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
