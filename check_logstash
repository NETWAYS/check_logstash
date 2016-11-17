#!/usr/bin/env ruby
#
# File : check_logstash
# Author : Thomas Widhalm, Netways
# E-Mail: thomas.widhalm@netways.de
# Date : 31/10/2016
#
# Version: 0.6.1-1
#
# This program is free software; you can redistribute it or modify
# it under the terms of the GNU General Public License version 3.0
#
# Changelog:
#       - 0.6.1 rewrite for better coding standards and rspec tests
# 	- 0.6.0 first stable release, working with first Logstash 5.0 release
# 	- 0.5.0 first beta, working with Logstash 5.0 release
# 	- 0.4.0 add thresholds and performance data for inflight events
#	- 0.3.0 change thresholds to adopt plugin syntax, change api calls for
#	  Logstash v5-alpha5
# 	- 0.2.0 add heap and file descriptor thresholds
# 	- 0.1.0 initial, untested prototype
#
#
# Plugin check for icinga
#
# Acknowledgements:
# 	A big "Thank you, you're awesome!" to Jordan Sissel who used a pull
# 	request to replace almost all of my code for version 0.2.0 and made
# 	this so much better. I hope I can keep up to the expectations with
# 	the next versions.

require 'rubygems'
require 'json'
require 'net/http'
require 'uri'
require 'optparse'

class CheckLogstash
  class Status
    attr_reader :message
    def initialize(message)
      @message = message
    end

    def label
      # get the class name like 'CheckLogstash::Critical' as 'CRITICAL'
      self.class.name.upcase.split('::').last
    end

    def to_s
      "#{label}: #{message}"
    end
  end

  class Critical < Status
    def to_i
      2
    end
  end

  class Warning < Status
    def to_i
      1
    end
  end

  class OK < Status
    def to_i
      0
    end
  end

  module CLI

    module_function

    def run(args)
      check = CheckLogstash.new
      parse(check, args)

      # fetch the result
      result = check.fetch
      health = check.health(result)

      # Get the maximum status code (Critical > Warning > OK)
      code = health.collect(&:to_i).max

      bad = health.any? { |h| h.is_a?(Critical) || h.is_a?(Warning) }

      status = if bad
        # If logstash is unhealthy, note it and include the first health report
        # that is bad.
        if health.any? { |h| h.is_a?(Critical) }
          "CRITICAL - Logstash is unhealthy - #{health.find { |h| h.is_a?(Critical) }}"
        elsif health.any? { |h| h.is_a?(Warning) }
          "WARNING - Logstash may not be healthy - #{health.find { |h| h.is_a?(Warning) }}"
        end
        else
          'OK - Logstash seems to be doing fine.'
        end

      puts "#{status} | #{check.performance_data(result)}\n"
      puts health.sort_by(&:to_i).reverse.join("\n")

      code
    end

    def parse(check, args)
      args = ['-h'] if args.empty?

      OptionParser.new do |opts|
        opts.banner = "Usage: #{$PROGRAM_NAME} [options]"

        options_error = proc do |message|
          $stderr.puts message
          $stderr.puts opts
          exit 3
        end

        opts.on('-H', '--hostname HOST', 'Logstash host') { |v| check.host = v }
        opts.on('-p', '--hostname PORT', 'Logstash API port') { |v| check.port = v.to_i }
        opts.on('--file-descriptor-threshold-warn WARN', 'The percentage relative to the process file descriptor limit on which to be a warning result.') { |v| check.warning_file_descriptor_percent = v.to_i }
        opts.on('--file-descriptor-threshold-crit CRIT', 'The percentage relative to the process file descriptor limit on which to be a critical result.') { |v| check.critical_file_descriptor_percent = v.to_i }
        opts.on('--heap-usage-threshold-warn WARN', 'The percentage relative to the heap size limit on which to be a warning result.') { |v| check.warning_heap_percent = v.to_i }
        opts.on('--heap-usage-threshold-crit CRIT', 'The percentage relative to the heap size limit on which to be a critical result.') { |v| check.critical_heap_percent = v.to_i }
        opts.on('--cpu-usage-threshold-warn WARN', 'The percentage of CPU usage on which to be a warning result.') { |v| check.warning_cpu_percent = v.to_i }
        opts.on('--cpu-usage-threshold-crit CRIT', 'The percentage of CPU usage on which to be a critical result.') { |v| check.critical_cpu_percent = v.to_i }
        # the following 2 blocks split : seperated ranges into 2 values. If only one value is given it's used as maximum
        opts.on('--inflight-events-warn WARN', 'Threshold for inflight events to be a warning result. Use min:max for a range.') do |v|
          options_error.call('--inflight-events-warn requires an argument') if v.nil?

          values = v.split(':')
          options_error.call("--inflight-events-warn has invalid argument #{v}") if values.count.zero? || values.count > 2

          begin
            if values.count == 1
              check.warning_inflight_events_min = -1
              check.warning_inflight_events_max = values[0].to_i
            else
              check.warning_inflight_events_min = values[0].to_i
              check.warning_inflight_events_max = values[1].to_i
            end
          rescue ArgumentError => e
            options_error.call("--inflight-events-warn has invalid argument. #{e.message}")
          end
        end
        opts.on('--inflight-events-crit CRIT', 'Threshold for inflight events to be a critical result. Use min:max for a range.') do |v|
          options_error.call('--inflight-events-critical requires an argument') if v.nil?

          values = v.split(':')
          options_error.call("--inflight-events-critical has invalid argument #{v}") if values.count.zero? || values.count > 2

          begin
            if values.count == 1
              check.critical_inflight_events_min = -1
              check.critical_inflight_events_max = values[0].to_i
            else
              check.critical_inflight_events_min = values[0].to_i
              check.critical_inflight_events_max = values[1].to_i
            end
          rescue ArgumentError => e
            options_error.call("--inflight-events-crit has invalid argument. #{e.message}")
          end
        end

        opts.on_tail('-h', '--help', 'Show this message') do
          puts opts
          exit
        end
      end.parse(args)
    end
  end # module CLI

  class Result
    class InvalidField < StandardError; end

    def initialize(data)
      @data = data
    end

    def self.from_hash(data)
      new(data)
    end

    # Provide dot-notation for querying a given field in a hash
    def get(field)
      self.class.get(field, @data)
    end

    def self.get(field, data)
      first, remaining = field.split('.', 2)
      value = data.fetch(first)
      if value.is_a?(Hash) && remaining
        get(remaining, value)
      else
        value
      end
    rescue KeyError
      raise InvalidField, field
    end
  end

  module Fetcher

    module_function

    def fetch(host, port)
      uri = URI.parse("http://#{host}:#{port}/_node/stats")
      http = Net::HTTP.new(uri.host, uri.port)
      request = Net::HTTP::Get.new(uri.request_uri)
      response = http.request(request)

      critical("Got HTTP response #{response.code}") if response.code != '200'

      result = begin
        JSON.parse(response.body)
      rescue => e
        critical("Failed parsing JSON response. #{e.class.name}")
      end
      Result.from_hash(result)
    end
  end

  module PerfData
    # return perfdata formatted string
    # use for values taken directly from API

    module_function

    def report(result, field, warning = nil, critical = nil, minimum = nil, maximum = nil)
      # 'label'=value[UOM];[warn];[crit];[min];[max]
      format('%s=%s;%s;%s;%s;%s', field, result.get(field), warning, critical, minimum, maximum)
    end

    def report_counter(result, field, warning = nil, critical = nil, minimum = nil, maximum = nil)
      # 'label'=value[UOM];[warn];[crit];[min];[max]
      # the UOM (unit of measurement) of 'c' means a counter.
      format('%s=%sc;%s;%s;%s;%s', field, result.get(field), warning, critical, minimum, maximum)
    end

    def report_percent(result, field, warning = nil, critical = nil, minimum = nil, maximum = nil)
      # 'label'=value[UOM];[warn];[crit];[min];[max]
      # the UOM (unit of measurement) of '%' means percent
      format('%s=%s%%;%s;%s;%s;%s', field, result.get(field), warning, critical, minimum, maximum)
    end
  end

  module PerfData_derived
    # return perfdata formatted string
    # use for derived / computed values

    module_function

    def report(label, value, warning = nil, critical = nil, minimum = nil, maximum = nil)
      format('%s=%s;%s;%s;%s;%s', label, value, warning, critical, minimum, maximum)
    end
  end

  Version = '0.6.1-1'
  DEFAULT_PORT = 9600
  DEFAULT_HOST = '127.0.0.1'

  DEFAULT_FILE_DESCRIPTOR_WARNING = 85
  DEFAULT_FILE_DESCRIPTOR_CRITICAL = 95
  DEFAULT_HEAP_WARNING = 70
  DEFAULT_HEAP_CRITICAL = 80
  DEFAULT_CPU_WARNING = nil
  DEFAULT_CPU_CRITICAL = nil
  DEFAULT_INFLIGHT_EVENTS_WARNING_MIN = nil
  DEFAULT_INFLIGHT_EVENTS_WARNING_MAX = nil
  DEFAULT_INFLIGHT_EVENTS_CRITICAL_MIN = nil
  DEFAULT_INFLIGHT_EVENTS_CRITICAL_MAX = nil

  attr_accessor :host, :port
  attr_accessor :warning_file_descriptor_percent
  attr_accessor :critical_file_descriptor_percent
  attr_accessor :warning_heap_percent
  attr_accessor :critical_heap_percent
  attr_accessor :warning_cpu_percent
  attr_accessor :critical_cpu_percent
  attr_accessor :warning_inflight_events_min
  attr_accessor :warning_inflight_events_max
  attr_accessor :critical_inflight_events_min
  attr_accessor :critical_inflight_events_max

  def initialize
    @host = DEFAULT_HOST
    @port = DEFAULT_PORT

    self.warning_file_descriptor_percent = DEFAULT_FILE_DESCRIPTOR_WARNING
    self.critical_file_descriptor_percent = DEFAULT_FILE_DESCRIPTOR_CRITICAL
    self.warning_heap_percent = DEFAULT_HEAP_WARNING
    self.critical_heap_percent = DEFAULT_HEAP_CRITICAL
    self.warning_cpu_percent = DEFAULT_CPU_CRITICAL
    self.critical_cpu_percent = DEFAULT_CPU_CRITICAL
    self.warning_inflight_events_min = DEFAULT_INFLIGHT_EVENTS_WARNING_MIN
    self.warning_inflight_events_max = DEFAULT_INFLIGHT_EVENTS_WARNING_MAX
    self.critical_inflight_events_min = DEFAULT_INFLIGHT_EVENTS_CRITICAL_MIN
    self.critical_inflight_events_max = DEFAULT_INFLIGHT_EVENTS_CRITICAL_MAX
  end

  def warning_file_descriptor_percent=(value)
    raise ArgumentError, "#{value} is not in range 0..100" unless (0..100).include?(value) || value.nil?
    @warning_file_descriptor_percent = value
  end

  def critical_file_descriptor_percent=(value)
    raise ArgumentError, "#{value} is not in range 0..100" unless (0..100).include?(value) || value.nil?
    @critical_file_descriptor_percent = value
  end

  def warning_heap_percent=(value)
    raise ArgumentError, "#{value} is not in range 0..100" unless (0..100).include?(value) || value.nil?
    @warning_heap_percent = value
  end

  def critical_heap_percent=(value)
    raise ArgumentError, "#{value} is not in range 0..100" unless (0..100).include?(value) || value.nil?
    @critical_heap_percent = value
  end

  def fetch
    Fetcher.fetch(host, port)
  end

  def performance_data(result)
    max_file_descriptors = result.get('process.max_file_descriptors')
    open_file_descriptors = result.get('process.open_file_descriptors')
    percent_file_descriptors = (open_file_descriptors.to_f / max_file_descriptors) * 100
    warn_file_descriptors = (max_file_descriptors / 100) * warning_file_descriptor_percent
    crit_file_descriptors = (max_file_descriptors / 100) * critical_file_descriptor_percent
    inflight_events = (result.get('pipeline.events.out') - result.get('pipeline.events.in')).to_i

    [
      PerfData.report_percent(result, 'process.cpu.percent', warning_cpu_percent, critical_cpu_percent, 0, 100),
      PerfData.report_percent(result, 'jvm.mem.heap_used_percent', warning_heap_percent, critical_heap_percent, 0, 100),
      PerfData.report(result, 'jvm.threads.count', nil, nil, 0, nil),
      PerfData.report(result, 'process.open_file_descriptors', warn_file_descriptors, crit_file_descriptors, 0, max_file_descriptors),
      # PerfData.report_counter(result, "pipeline.events.in", nil, nil, 0, nil),
      # PerfData.report_counter(result, "pipeline.events.filtered", nil, nil, 0, nil),
      PerfData.report_counter(result, 'pipeline.events.out', nil, nil, 0, nil),
      PerfData_derived.report('inflight_events', inflight_events, warning_inflight_events_max, critical_inflight_events_max, 0, nil)
      # inflight_perfdata
    ].join(' ')
  end

  # the reports are defined below, call them here

  def health(result)
    [
      file_descriptor_health(result),
      heap_health(result),
      inflight_events_health(result),
      config_reload_health(result),
      cpu_usage_health(result)
    ]
  end

  # reports for various performance data including threshold checks

  FILE_DESCRIPTOR_REPORT = 'Open file descriptors at %.2f%%. (%d out of %d file descriptors are open)'
  def file_descriptor_health(result)
    max_file_descriptors = result.get('process.max_file_descriptors')
    open_file_descriptors = result.get('process.open_file_descriptors')

    # For now, we have to compute the file descriptor usage percent until
    # Logstash stats api delivers the percentage directly.
    percent_file_descriptors = (open_file_descriptors.to_f / max_file_descriptors) * 100.0

    file_descriptor_report = format(FILE_DESCRIPTOR_REPORT, percent_file_descriptors, open_file_descriptors, max_file_descriptors)

    if critical_file_descriptor_percent && percent_file_descriptors > critical_file_descriptor_percent
      Critical.new(file_descriptor_report)
    elsif warning_file_descriptor_percent && percent_file_descriptors > warning_file_descriptor_percent
      Warning.new(file_descriptor_report)
    else
      OK.new(file_descriptor_report)
    end
  end

  HEAP_REPORT = 'Heap usage at %.2f%% (%d out of %d bytes in use)'
  def heap_health(result)
    percent_heap_used = result.get('jvm.mem.heap_used_percent')
    heap_report = format(HEAP_REPORT, percent_heap_used, result.get('jvm.mem.heap_used_in_bytes'), result.get('jvm.mem.heap_max_in_bytes'))

    if critical_heap_percent && percent_heap_used > critical_heap_percent
      Critical.new(heap_report)
    elsif warning_heap_percent && percent_heap_used > warning_heap_percent
      Warning.new(heap_report)
    else
      OK.new(heap_report)
    end
  end

  INFLIGHT_EVENTS_REPORT = 'Inflight events: %d'
  def inflight_events_health(result)
    # check if inflight events are outside of threshold
    # find a way to reuse the already computed inflight events
    inflight_events = (result.get('pipeline.events.out') - result.get('pipeline.events.in')).to_i
    inflight_events_report = format(INFLIGHT_EVENTS_REPORT, inflight_events)
    if critical_inflight_events_max && critical_inflight_events_max < inflight_events
      Critical.new(inflight_events_report)
    elsif critical_inflight_events_min && critical_inflight_events_min > inflight_events
      Critical.new(inflight_events_report)
    elsif warning_inflight_events_max && warning_inflight_events_max < inflight_events
      Warning.new(inflight_events_report)
    elsif warning_inflight_events_min && warning_inflight_events_min > inflight_events
      Warning.new(inflight_events_report)
    else
      OK.new(inflight_events_report)
    end
  end

  # the following would be needed to output the whole errormessage
  # CONFIG_RELOAD_REPORT = "Config reload errormessage: %s"
  CONFIG_RELOAD_REPORT = 'Config reload syntax check' 
  def config_reload_health(result)
    config_reload_errors = (result.get('pipeline.reloads.failures')).to_i
    config_reload_error_message = (result.get('pipeline.reloads.last_error.message'))
    config_reload_errors_report = format(CONFIG_RELOAD_REPORT, config_reload_error_message)
    # the following would output the whole errormessage which is too long as output of a monitoring plugin
    # config_reload_errors_report = format(CONFIG_RELOAD_REPORT, config_reload_error_message)
    if config_reload_errors > 0
      Critical.new(config_reload_errors_report)
    else
      OK.new(config_reload_errors_report)
    end
  end

  CPU_REPORT = 'CPU usage in percent: %d'
  def cpu_usage_health(result)
    cpu_usage_percent = (result.get('process.cpu.percent')).to_i
    cpu_usage_percent_report = format(CPU_REPORT, cpu_usage_percent)
    if critical_cpu_percent && cpu_usage_percent > critical_cpu_percent
      Critical.new(cpu_usage_percent_report)
    elsif warning_cpu_percent && cpu_usage_percent > warning_cpu_percent
      Warning.new(cpu_usage_percent_report)
    else
      OK.new(cpu_usage_percent_report)
    end
  end
 
end

if __FILE__ == $PROGRAM_NAME
  exit(CheckLogstash::CLI.run(ARGV))
end
