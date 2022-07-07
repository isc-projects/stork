
namespace :hook do
    desc "Init new hook directory
        DEST - the hook directory - required
        PACKAGE - the package name - required"
    task :init do
        destination = ENV["DEST"]
        package = ENV["PACKAGE"]
        if destination.nil?
            fail "You must provide the DEST variable with the hook directory"
        end
        if package.nil?
            fail "You must provide the PACKAGE variable with the package name"
        end

        require 'pathname'
        main_package = "isc.org/stork@v0.0.0"
        main_package_directory_abs = Pathname.new('backend').realdirpath
        package_directory_abs = Pathname.new('destination').realdirpath
        package_directory_rel = package_directory_abs.relative_path_from main_package_directory_abs

        sh "mkdir", "-p", destination
        Dir.chdir(destination) do
            
            sh GO, "mod", "init", package
            sh GO, "mod", "edit", "-require", main_package
            sh GO, "mod", "edit", "-replace", "#{main_package}=#{package_directory_rel}"
        end
    end

    desc "Build the hook
        DEST - the hook directory - required"
    task :build do
        destination = ENV["DEST"]
        if destination.nil?
            fail "You must provide the DEST variable with the hook directory"
        end

        Dir.chdir(destination) do
            sh GO, "mod", "tidy"
            sh GO, "build", "-buildmode=plugin"
        end
    end 
end

