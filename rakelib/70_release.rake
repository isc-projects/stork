# Release
# This file contains the release-stage tasks.

namespace :release do
    desc 'Generic task for bumping up version
        VERSION - target version after bump - required
    '
    task :bump do
        if ENV["VERSION"].nil?
            fail "Environment variable VERSION is not specified"
        end

        # Replace version in all files. Use specific patterns for each for stricter matching.
        for i in [
            ['api/swagger.in.yaml', "version: #{STORK_VERSION}", "version: #{ENV["VERSION"]}"],
            ['backend/version.go', "const Version = \"#{STORK_VERSION}\"", "const Version = \"#{ENV["VERSION"]}\""],
            ['webui/package.json', "\"version\": \"#{STORK_VERSION}\"", "\"version\": \"#{ENV["VERSION"]}\""],
            ['webui/package-lock.json', "\"version\": \"#{STORK_VERSION}\"", "\"version\": \"#{ENV["VERSION"]}\""],
        ] do
            File.open(i[0], 'r') do |file_r|
                contents = file_r.read
                contents.sub!(i[1], i[2])
                File.open(i[0], 'w') do |file_w|
                    file_w.write(contents)
                end
            end
        end

        # Prepend the unreleased ChangeLog entries to the ChangeLog file.
        Rake::Task['release:changelog'].invoke()

        # Announce release in ChangeLog.
        days_to_add = (3 + 7 - Date.today.wday) % 7
        next_wednesday = Date.today + days_to_add
        File.open('ChangeLog.md', 'r') do |file_r|
            contents = file_r.read
            contents = "Stork #{ENV["VERSION"]} released on #{next_wednesday}.\n\n" + contents

            File.write('ChangeLog.md', contents)
        end

        # Put out an informative message that the bump was successful.
        puts "Version succesfully bumped to #{ENV["VERSION"]}."
    end


    namespace :bump do
        desc 'Bump up major version'
        task :major do
            major = STORK_VERSION.split('.')[0]
            major = Integer(major) + 1
            ENV["VERSION"] = "#{major}.0.0"
            Rake::Task['release:bump'].invoke()
        end

        desc 'Bump up minor version'
        task :minor do
            major = STORK_VERSION.split('.')[0]
            minor = STORK_VERSION.split('.')[1]
            minor = Integer(minor) + 1
            ENV["VERSION"] = "#{major}.#{minor}.0"
            Rake::Task['release:bump'].invoke()
        end

        desc 'Bump up patch version'
        task :patch do
            major = STORK_VERSION.split('.')[0]
            minor = STORK_VERSION.split('.')[1]
            patch = STORK_VERSION.split('.')[2]
            patch = Integer(patch) + 1
            ENV["VERSION"] = "#{major}.#{minor}.#{patch}"
            Rake::Task['release:bump'].invoke()
        end
    end

    desc 'Merge unreleased ChangeLog entries to the ChangeLog file'
    task :changelog do
        # This script reads the ChangeLog entries from the "changelog_unreleased/"
        # directory. It fills up the number of entries, concats them by a blank
        # line and prepends to the ChangeLog.md file.

        changelog_dir = 'changelog_unreleased'
        changelog_file = 'ChangeLog.md'

        # Read the ChangeLog file content.
        changelog_content = File.read(changelog_file)

        # Get the start number of the entries by iterating over the ChangeLog lines.
        start = 0
        changelog_content.each_line do |line|
            # Seach for first entry header. Skip the release header.
            if line.start_with? '*'
                start = Integer(line.split(' ')[1])
                break
            end
        end

        # List files in the changelog directory.
        entry_filenames = Dir.entries(changelog_dir)
        entry_files_and_timestamps = []

        entry_filenames.each do |entry_filename|
            entry_file = File.join(changelog_dir, entry_filename)
            # Filter out non-files.
            if !File.file?(entry_file)
                next
            end

            # Skip hidden files.
            if entry_filename.start_with? '.'
                next
            end

            # Fetch the git committer date as the unix timestamp.
            stdout, status = Open3.capture2 "git", "log", "-1", "--format=%ct", entry_file
            timestamp = 0
            if status == 0
                timestamp = Integer(stdout)
            end

            # Append entry to the list.
            entry_files_and_timestamps.push([entry_file, timestamp])
        end

        # Sort the entries by the timestamp. Oldest first.
        entry_files_and_timestamps.sort_by! { |e| e[1] }

        # Extract the file names from the list.
        entry_files = entry_files_and_timestamps.map { |e| e[0] }

        # Iterate over the entry files.
        entries = []
        entry_files.each do |entry_file|
            # Read the file content.
            entry_content = File.read(entry_file)

            # Strip the leading asterisk from the beginning of the file.
            entry_content.sub!(/^\s*\*/, '')

            # Normalize the indentation to 4 spaces. Preserve the empty lines.
            entry_content.gsub!(/^[ \t]*([^ \t])/, '    \1')
            entry_content.gsub!(/^    $/, '')

            # Strip the whitespace from the beginning and the end of the file.
            entry_content.strip!

            # Increment the start number.
            start += 1

            # Prepend the entry with the number.
            entry_content = "* #{start} #{entry_content}"

            # Append the entry to the list.
            entries.push(entry_content)
        end

        # Return if there are no entries.
        if entries.length == 0
            puts "No entries to merge."
            next
        end

        # Reverse the entries.
        entries.reverse!

        # Concat the entries.
        merged = entries.join("\n\n")

        # Concat the new entries and current ChangeLog content.
        merged = merged + "\n\n" + changelog_content

        # Write the new ChangeLog content.
        File.write(changelog_file, merged)

        # Remove the files in the changelog directory.
        entry_files.each do |entry_file|
            File.delete(entry_file)
        end
    end

    desc 'Prepare release notes'
    task :notes => [WGET, SED, PERL, FOLD] do
        release_notes_filename = "Stork-#{STORK_VERSION}-ReleaseNotes.txt"
        release_notes_filename_in = release_notes_filename + ".in"
        release_notes_file = File.new(release_notes_filename, 'w' )

        at_exit {
          sh 'rm', '-f', release_notes_filename_in
        }

        fetch_file("https://gitlab.isc.org/isc-projects/stork/-/wikis/Releases/Release-notes-#{STORK_VERSION}.md", release_notes_filename_in)

        Open3.pipeline [
            "cat", release_notes_filename_in
        ], [
            # Removes the triple backticks.
            SED, '/^```/d'
        ], [
            # Removes backslashes prepending square brackets.
            SED, 's/\\\[/[/g;s/\\\]/]/g'
        ], [
            # Replaces square brackets with round brackets for hyperlinks.
            PERL, '-pe', 's|\[(http.*?)\]\(http.*\)|\1|',
        ], [
            # Wrap rows to width 73 == 72 + newline. Historically, number 72 has something to do with punch cards.
            FOLD, '-sw', '73'
        ], [
            # Remove trailing blank spaces.
            SED, 's/ *$//g'
        ], :out => release_notes_file

    end

    desc 'Prepare release tarball with Stork sources'
    task :tarball => [GIT] do
        sh GIT, "archive",
            "--prefix", "stork-#{STORK_VERSION}/",
            "-o", "stork-#{STORK_VERSION}.tar.gz", "HEAD"
    end
    CLEAN.append *FileList["stork-*.tar.gz"]


    namespace :tarball do
        desc 'Upload tarball(s) and release notes to given host and path
            HOST - the SSH host - required
            TARGET - the target path for tarball file - required'
        task :upload => [SSH, SCP] do
            host = ENV["HOST"]
            target = ENV["TARGET"]
            if host.nil?
                fail "You need to provide the HOST variable"
            elsif target.nil?
                fail "You need to provide the TARGET variable"
            end

            path = "#{target}/#{STORK_VERSION}"
            sh SSH, host, "--", "mkdir", "-p", path
            sh SCP, "-p",
                       *FileList["./stork*-#{STORK_VERSION}.tar.gz"],
                       "./Stork-#{STORK_VERSION}-ReleaseNotes.txt",
                       "#{host}:#{path}"
            sh SSH, host, "--", "chmod", "-R", "g+w", path
        end
    end

    namespace :packages do
        desc 'Upload packages to Cloudsmith
            CS_API_KEY - the Cloudsmith API key - required
            COMPONENTS - the filename component - required
            REPO - the Cloudsmith repository - auto-determined if not provided'
        task :upload => [CLOUDSMITH] do
            key = ENV["CS_API_KEY"]
            repo = ENV["REPO"]
            components = ENV["COMPONENTS"]
            if key.nil?
                fail "You need to provide the CS_API_KEY variable"
            elsif components.nil?
                fail "You need to provide the COMPONENTS variable"
            end

            sh CLOUDSMITH, "check", "service"
            sh CLOUDSMITH, "whoami", "-k", "#{key}"

            package_type_commands = {
                'deb' => 'deb',
                'rpm' => 'rpm',
                'apk' => 'alpine',
            }

            for package_type in ['deb', 'rpm', 'apk'] do
                type_command = package_type_commands[package_type]
                distro = 'any-distro'
                if package_type == 'apk'
                    distro = 'alpine'
                end

                for component in components.split(",") do
                    component = component.strip
                    pattern = component + '*\.' + package_type
                    files = Dir.glob(pattern)
                    if files.nil? || files.length == 0
                        fail 'ERROR: could not find any files matching ' + pattern
                    end
                    files.each do |file|
                        if repo.nil? then
                            versions = file.match(/^\D*(\d+)\.(\d+)\.(\d+)/)
                            stork_minor_version = versions[2]
                            is_development_version = Integer(stork_minor_version) % 2 == 1
                            if is_development_version then
                                repo = 'stork-dev'
                            else
                                repo = 'stork'
                            end
                        end
                        sh CLOUDSMITH, "upload", type_command, "-k", "#{key}", "-W", "--republish", "isc/#{repo}/#{distro}/any-version", file
                    end
                end
            end
        end
    end
end

namespace :check do
    desc 'Check the external dependencies related to the distribution'
    task :release do
        check_deps(__FILE__)
    end
end
