TOOLS_DIR = File.expand_path('tools')
NODE_VER = 'node-v10.16.3-linux-x64'
ENV['PATH'] = "#{TOOLS_DIR}/#{NODE_VER}/bin:#{ENV['PATH']}"
NPX = "#{TOOLS_DIR}/#{NODE_VER}/bin/npx"
SWAGGER_CODEGEN = "#{TOOLS_DIR}/swagger-codegen-cli-2.4.8.jar"
GOSWAGGER = "#{TOOLS_DIR}/swagger_linux_amd64"
SWAGGER_FILE = File.expand_path('api/swagger.yaml')
NG = File.expand_path('webui/node_modules/.bin/ng')
ENV['PATH'] = "#{TOOLS_DIR}/go/bin:#{ENV['PATH']}"
GO = "#{TOOLS_DIR}/go/bin/go"
GOLANGCILINT = "#{TOOLS_DIR}/golangci-lint-1.19.1-linux-amd64/golangci-lint"


# SERVER
file GO do
  sh "mkdir -p $HOME/go"
  sh "mkdir -p #{TOOLS_DIR}"
  Dir.chdir(TOOLS_DIR) do
    sh 'wget https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz -O go.tar.gz'
    sh 'tar -zxf go.tar.gz'
  end
end

task :gen_server => [GO, GOSWAGGER] do
  Dir.chdir('backend/server') do
    sh "#{GOSWAGGER} generate server --target gen --name Stork --spec #{SWAGGER_FILE}"
  end
end

file GOSWAGGER do
  sh "mkdir -p #{TOOLS_DIR}"
  sh "wget https://github.com/go-swagger/go-swagger/releases/download/v0.20.1/swagger_linux_amd64 -O #{GOSWAGGER}"
  sh "chmod a+x #{GOSWAGGER}"
end

task :build_server => [:gen_server, GO] do
  sh "cd backend/server && #{GO} build -v gen/cmd/stork-server/main.go"
end

task :run_server => [:gen_server, GO] do
  sh "cd backend/server && #{GO} run gen/cmd/stork-server/main.go --port 8765"
end

file GOLANGCILINT do
  sh "mkdir -p #{TOOLS_DIR}"
  Dir.chdir(TOOLS_DIR) do
    sh "wget https://github.com/golangci/golangci-lint/releases/download/v1.19.1/golangci-lint-1.19.1-linux-amd64.tar.gz -O golangci-lint.tar.gz"
    sh "tar -zxf golangci-lint.tar.gz"
  end
end

task :lint_go => [GO, GOLANGCILINT, :gen_server] do
  Dir.chdir('backend/server') do
    sh 'echo $PATH'
    sh "#{GOLANGCILINT} run gen/restapi"
  end
end


# WEBUI
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

task :lint_ui => [NG, :gen_client] do
  Dir.chdir('webui') do
    sh 'npx ng lint'
  end
end

task :ci_ui => [:gen_client] do
  Dir.chdir('webui') do
    sh 'npm ci'
    sh 'npx ng lint'
#    sh 'CHROME_BIN=/usr/bin/chromium-browser npx ng test --progress false --watch false'
#    sh 'npx ng e2e --progress false --watch false'
  end
end


# DOCKER
task :docker_up => [:build_server, :build_ui] do
  sh "docker-compose up"
end

task :docker_down do
  sh "docker-compose down"
end


# OTHER
task :clean do
  sh "rm -rf #{TOOLS_DIR} server/main"
end

task :prepare_env => [GO, GOSWAGGER, GOLANGCILINT, SWAGGER_CODEGEN, NPX] do
  sh "mkdir -p $HOME/go"
end
