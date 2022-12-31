require 'rspec/core/rake_task'
#require 'rubocop/rake_task'

RSpec::Core::RakeTask.new(:spec)
#RuboCop::RakeTask.new

file "check_logstash" do
  puts "Creating the plugin"
  sh "cat lib/check_logstash.rb > check_logstash"
  sh "chmod +x check_logstash"
end

task default: [:spec, :check_logstash]
