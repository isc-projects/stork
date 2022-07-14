
namespace :hook do
    desc "Init new hook directory
        PACKAGE - the package name - required"
    task :init => [GO] do
        package = ENV["PACKAGE"]
        
        package_last_segment = package
        last_segment_idx = package.rindex "/"
        if !last_segment_idx.nil?
            package_last_segment = package[last_segment_idx+1..-1]
        end

        destination = File.join("plugins", package_last_segment)

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
            sh "git", "init"
            sh GO, "mod", "init", package
            sh GO, "mod", "edit", "-require", main_package
            sh GO, "mod", "edit", "-replace", "#{main_package}=#{package_directory_rel}"
        end
    end

    desc "Build the hooks
        DEBUG - build plugins in debug mode - default: false
    "
    task :build => [GO] do
        Dir.foreach("plugins") do |filename|
            path = File.join("plugins", filename)
            next if filename == '.' or filename == '..' or !File.directory? path

            flags = []
            if ENV["DEBUG"] == "true"
                flags.append "-gcflags", "all=-N -l"
            end

            Dir.chdir(path) do
                sh GO, "mod", "tidy"
                sh GO, "build", *flags, "-buildmode=plugin", "-o", ".."
            end
        end
    end 
end

