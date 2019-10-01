SWAGGER_FILE = "../swagger.yaml"
NG = "../node_modules/.bin/ng"

task :gen_server => 'backend/swagger_linux_amd64' do
  Dir.chdir('backend') do
    sh "./swagger_linux_amd64 generate server --target gen --name Stork --spec #{SWAGGER_FILE}"
  end
end

file 'backend/swagger_linux_amd64' do
  sh "wget https://github.com/go-swagger/go-swagger/releases/download/v0.20.1/swagger_linux_amd64 -O backend/swagger_linux_amd64"
  sh "chmod a+x backend/swagger_linux_amd64"
end

task :build_server => :gen_server do
  sh "cd backend && go build -v gen/cmd/stork-server/main.go"
end

task :clean do
  sh "rm -rf backend/swagger_linux_amd64"
end

task :run_server => :gen_server do
  sh "cd backend && go run gen/cmd/stork-server/main.go --port 8765"
end

task :gen_client => 'frontend/swagger-codegen-cli-2.4.8.jar' do
  Dir.chdir('frontend') do
    sh "java -jar swagger-codegen-cli-2.4.8.jar generate -l typescript-angular -i #{SWAGGER_FILE} -o src/app/backend --additional-properties snapshot=true,ngVersion=8.2.8"
  end
end

file 'frontend/swagger-codegen-cli-2.4.8.jar' do
  sh "wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.8/swagger-codegen-cli-2.4.8.jar -O frontend/swagger-codegen-cli-2.4.8.jar"
end

task :build_ui => :gen_client do
  Dir.chdir('frontend') do
    sh "#{NG} build --prod"
  end
end

task :docker_up => [:build_server, :build_ui] do
  sh "docker-compose up"
end

task :docker_down do
  sh "docker-compose down"
end
