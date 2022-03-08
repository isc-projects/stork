desc 'Generate ctags for Emacs'
task :ctags do
  sh 'etags.ctags -f TAGS -R --exclude=webui/node_modules --exclude=webui/dist --exclude=tools .'
end