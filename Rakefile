TOOLS_DIR = File.expand_path('tools')
NODE_VER = 'node-v10.16.3-linux-x64'
ENV['PATH'] = "#{TOOLS_DIR}/#{NODE_VER}/bin:#{ENV['PATH']}"
NPX = "#{TOOLS_DIR}/#{NODE_VER}/bin/npx"
SWAGGER_CODEGEN = "#{TOOLS_DIR}/swagger-codegen-cli-2.4.8.jar"
GOSWAGGER = "#{TOOLS_DIR}/swagger_linux_amd64"
SWAGGER_FILE = File.expand_path('swagger.yaml')
NG = File.expand_path('webui/node_modules/.bin/ng')

# SERVER
task :gen_server => GOSWAGGER do
  Dir.chdir('server') do
    sh "#{GOSWAGGER} generate server --target gen --name Stork --spec #{SWAGGER_FILE}"
  end
end

file GOSWAGGER do
  sh "mkdir -p #{TOOLS_DIR}"
  sh "wget https://github.com/go-swagger/go-swagger/releases/download/v0.20.1/swagger_linux_amd64 -O #{GOSWAGGER}"
  sh "chmod a+x #{GOSWAGGER}"
end

task :build_server => :gen_server do
  sh "cd server && go build -v gen/cmd/stork-server/main.go"
end

task :run_server => :gen_server do
  sh "cd server && go run gen/cmd/stork-server/main.go --port 8765"
end

# CLIENT
task :gen_client => SWAGGER_CODEGEN do
  Dir.chdir('webui') do
    sh "java -jar #{SWAGGER_CODEGEN} generate -l typescript-angular -i #{SWAGGER_FILE} -o src/app/backend --additional-properties snapshot=true,ngVersion=8.2.8"
  end
end

file SWAGGER_CODEGEN do
  sh "mkdir -p #{TOOLS_DIR}"
  sh "wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.8/swagger-codegen-cli-2.4.8.jar -O #{SWAGGER_CODEGEN}"
end

file NPX do
  sh "mkdir -p #{TOOLS_DIR}"
  Dir.chdir(TOOLS_DIR) do
    sh "wget https://nodejs.org/dist/v10.16.3/#{NODE_VER}.tar.xz -O #{TOOLS_DIR}/node.tar.xz"
    sh "tar -Jxf node.tar.xz"
  end
end

file NG => NPX do
  Dir.chdir('webui') do
    sh 'npm install'
  end
end

task :build_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx ng build --prod'
  end
end

task :serve_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx ng serve'
  end
end

# DOCKER
task :docker_up => [:build_server, :build_ui] do
  sh "docker-compose up"
end

task :docker_down do
  sh "docker-compose down"
end

task :clean do
  sh "rm -rf #{TOOLS_DIR} server/main"
end
