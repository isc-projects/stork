# establish Stork version
stork_version = '0.0.0'
version_file = 'backend/version.go'
text = File.open(version_file).read
text.each_line do |line|
  if line.start_with? 'const Version'
    parts = line.split('"')
    stork_version = parts[1]
  end
end
STORK_VERSION = stork_version

desc 'Prepare release tarball with Stork sources'
task :tarball do
    sh "git", "archive",
        "--prefix", "stork-#{STORK_VERSION}/",
        "-o", "stork-#{STORK_VERSION}.tar.gz", "HEAD"
end
CLEAN.append *FileList["stork-*.tar.gz"]

desc 'Generic task for bumping up version'
task :bump, [:version] do |t, args|
  # Replace version in all files. Use specific patterns for each for stricter matching.
  for i in [
      ['api/swagger.in.yaml', "version: #{STORK_VERSION}", "version: #{args[:version]}"],
      ['backend/version.go', "const Version = \"#{STORK_VERSION}\"", "const Version = \"#{args[:version]}\""],
      ['webui/package.json', "\"version\": \"#{STORK_VERSION}\"", "\"version\": \"#{args[:version]}\""],
      ['webui/package-lock.json', "\"version\": \"#{STORK_VERSION}\"", "\"version\": \"#{args[:version]}\""],
  ] do
    File.open(i[0], 'r') do |file_r|
      contents = file_r.read
      contents.sub!(i[1], i[2])
      File.open(i[0], 'w') do |file_w|
        file_w.write(contents)
      end
    end
  end

  # Announce release in ChangeLog.
  days_to_add = (3 + 7 - Date.today.wday) % 7
  next_wednesday = Date.today + days_to_add
  File.open('ChangeLog.md', 'r') do |file_r|
    contents = file_r.read
    contents = "Stork #{args[:version]} released on #{next_wednesday}.\n\n" + contents
    File.open('ChangeLog.md', 'w') do |file_w|
      file_w.write(contents)
    end
  end

  # Put out an informative message that the bump was successful.
  puts "Version succesfully bumped to #{args[:version]}."
end

desc 'Bump up major version'
task :bump_major do
  major = STORK_VERSION.split('.')[0]
  major = Integer(major) + 1
  Rake::Task['bump'].invoke("#{major}.0.0")
end

desc 'Bump up minor version'
task :bump_minor do
  major = STORK_VERSION.split('.')[0]
  minor = STORK_VERSION.split('.')[1]
  minor = Integer(minor) + 1
  Rake::Task['bump'].invoke("#{major}.#{minor}.0")
end

desc 'Bump up patch version'
task :bump_patch do
  major = STORK_VERSION.split('.')[0]
  minor = STORK_VERSION.split('.')[1]
  patch = STORK_VERSION.split('.')[2]
  patch = Integer(patch) + 1
  Rake::Task['bump'].invoke("#{major}.#{minor}.#{patch}")
end
