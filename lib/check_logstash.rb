#!/usr/bin/env ruby
#
# File : check_logstash
# Author : Thomas Widhalm, Netways
# E-Mail: thomas.widhalm@netways.de
# Date : 18/07/2022
#
# Version: 0.8.0-0
#
# This program is free software; you can redistribute it or modify
# it under the terms of the GNU General Public License version 3.0
#
# Changelog:
#   - 0.8.0 Add option to check incoming/outgoing events per minute
#   - 0.7.4 Add option for checking only one pipeline + better handling for threshold ranges
#   - 0.7.3 fix inflight event calculation
#   - 0.7.2 fix handling of xpack-monitoring pipeline
#   - 0.7.1 fix multipipeline checks, improve errorhandling
#   - 0.6.2 update for multipipeline output
#   - 0.6.1 rewrite for better coding standards and rspec tests
#   - 0.6.0 first stable release, working with first Logstash 5.0 release
#   - 0.5.0 first beta, working with Logstash 5.0 release
#   - 0.4.0 add thresholds and performance data for inflight events
#   - 0.3.0 change thresholds to adopt plugin syntax, change api calls for
#	Logstash v5-alpha5
#   - 0.2.0 add heap and file descriptor thresholds
#   - 0.1.0 initial, untested prototype
#
#
# Plugin check for icinga
#
# Acknowledgements:
# 	A big "Thank you, you're awesome!" to Jordan Sissel who used a pull
# 	request to replace almost all of my code for version 0.2.0 and made
# 	this so much better. I hope I can keep up to the expectations with
# 	the next versions.
#
# 	Thank you very much to GitHub user rlueckl for helping out with
# 	getting it to work with Logstash 5 and 6

require 'rubygems'
require 'json'
require 'net/http'
require 'uri'
require 'optparse'
require 'time'

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

      # fetch the result and load saved state
      result = check.fetch
      state = check.load_state
      health = check.health(result, state)
      check.save_events_state(result)

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

      puts "#{status} | #{check.performance_data(result, state)}\n"
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
        opts.on('-P', '--pipeline PIPELINE', 'Pipeline to monitor, uses all pipelines when not set') { |v| check.pipeline = v }
        opts.on('--file-descriptor-threshold-warn WARN', 'The percentage relative to the process file descriptor limit on which to be a warning result.') { |v| check.warning_file_descriptor_percent = v.to_i }
        opts.on('--file-descriptor-threshold-crit CRIT', 'The percentage relative to the process file descriptor limit on which to be a critical result.') { |v| check.critical_file_descriptor_percent = v.to_i }
        opts.on('--heap-usage-threshold-warn WARN', 'The percentage relative to the heap size limit on which to be a warning result.') { |v| check.warning_heap_percent = v.to_i }
        opts.on('--heap-usage-threshold-crit CRIT', 'The percentage relative to the heap size limit on which to be a critical result.') { |v| check.critical_heap_percent = v.to_i }
        opts.on('--cpu-usage-threshold-warn WARN', 'The percentage of CPU usage on which to be a warning result.') { |v| check.warning_cpu_percent = v.to_i }
        opts.on('--cpu-usage-threshold-crit CRIT', 'The percentage of CPU usage on which to be a critical result.') { |v| check.critical_cpu_percent = v.to_i }
        opts.on('--temp-filedir NAME', 'Directory to use for the temporary state file. Only used when one of the events-per-minute metrics is used. Defaults to /tmp') { |v| check.temp_file_dir = v }
        # the following blocks split : seperated ranges into 2 values. If only one value is given it's used as maximum
        opts.on('--inflight-events-warn WARN', 'Threshold for inflight events to be a warning result. Use min:max for a range.') do |v|
          check.warning_inflight_events_min, check.warning_inflight_events_max = parse_min_max_option('inflight-events-warn', v, options_error)
        end
        opts.on('--inflight-events-crit CRIT', 'Threshold for inflight events to be a critical result. Use min:max for a range.') do |v|
          check.critical_inflight_events_min, check.critical_inflight_events_max = parse_min_max_option('inflight-events-crit', v, options_error)
        end
        opts.on('--events-in-per-minute-warn WARN', 'Threshold for the number of ingoing events per minute to be a warning. Use min:max for a range.') do |v|
          check.warning_events_in_per_minute_min, check.warning_events_in_per_minute_max = parse_min_max_option('events-in-per-minute-warn', v, options_error)
        end
        opts.on('--events-in-per-minute-crit CRIT', 'Threshold for the number of ingoing events per minute to be critical. Use min:max for a range.') do |v|
          check.critical_events_in_per_minute_min, check.critical_events_in_per_minute_max = parse_min_max_option('events-in-per-minute-crit', v, options_error)
        end
        opts.on('--events-out-per-minute-warn WARN', 'Threshold for the number of outgoing events per minute to be a warning. Use min:max for a range.') do |v|
          check.warning_events_out_per_minute_min, check.warning_events_out_per_minute_max = parse_min_max_option('events-out-per-minute-warn', v, options_error)
        end
        opts.on('--events-out-per-minute-crit CRIT', 'Threshold for the number of outgoing events per minute to be critical. Use min:max for a range.') do |v|
          check.critical_events_out_per_minute_min, check.critical_events_out_per_minute_max = parse_min_max_option('events-out-per-minute-crit', v, options_error)
        end

        opts.on_tail('-h', '--help', 'Show this message') do
          puts opts
          exit
        end
      end.parse(args)
    end

    def parse_min_max_option(parameter_name, v, options_error)
      options_error.call("--#{parameter_name} requires an argument") if v.nil?

      output_min = nil
      output_max = nil

      values = v.split(':')
      no_max = v[-1] == ':'
      options_error.call("--#{parameter_name} has invalid argument #{v}") if v[0] == ':' || values.count.zero? || values.count > 2

      begin
        if values.count == 1
          output_min = values[0].to_i if no_max
          output_max = values[0].to_i unless no_max
        else
          output_min = values[0].to_i
          output_max = values[1].to_i
        end
      rescue ArgumentError => e
        options_error.call("--#{parameter_name} has invalid argument. #{e.message}")
      end

      [output_min, output_max]
    end
  end # module CLI

  class Result
    class InvalidField < StandardError; end

    def initialize(data)
      @data = data
      @timestamp = Time.now.to_f
    end

    def self.from_hash(data)
      new(data)
    end

    def get_timestamp
      @timestamp
    end

    def has_key?(key)
      @data.has_key?(key)
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

    def critical(message)
      puts message
      exit(3)
    end

    def fetch(host, port, pipeline)
      uri = URI.parse("http://#{host}:#{port}/_node/stats")
      http = Net::HTTP.new(uri.host, uri.port)
      request = Net::HTTP::Get.new(uri.request_uri)
      response = http.request(request)

      critical("Got HTTP response #{response.code}") if response.code != '200'

      result = begin
        data = JSON.parse(response.body)
        data['pipelines'].select! {|p| p ==  pipeline} if pipeline
        data
      rescue => e
        critical("Failed parsing JSON response. #{e.class.name}")
      end
      critical("Pipeline not found: #{pipeline}") if pipeline && result['pipelines'].empty?
      Result.from_hash(result)
    end
  end

  module FileHandler
    # handles the data state using a temporary file.

    module_function

    def read(temp_file_dir, host, port, pipeline)
      temp_file_name = File.join(temp_file_dir, "check_logstash_#{host}_#{port}_#{pipeline || "all"}_events_state.tmp")
      return {} unless File.file?(temp_file_name)
      JSON.parse(File.read(temp_file_name))
    rescue Exception => e
      puts "Can not load state from temp file, reason: #{e}"
      exit(3)
    end

    def save(temp_file_dir, host, port, pipeline, data)
      temp_file_name = File.join(temp_file_dir, "check_logstash_#{host}_#{port}_#{pipeline || "all"}_events_state.tmp")
      File.write(temp_file_name, data.to_json)
    rescue Exception => e 
      puts "Can not save state to temp file, reason: #{e}"
      exit(3)
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

    def report(label, value, warning_min = nil, warning_max = nil, critical_min = nil, critical_max = nil)
      format('%s=%s;%s%s%s;%s%s%s', label, value, warning_min, warning_min ? ':' : '', warning_max, critical_min, critical_min ? ':' : '', critical_max)
    end
  end

  Version = '0.8.0-0'
  DEFAULT_PORT = 9600
  DEFAULT_HOST = '127.0.0.1'
  DEFAULT_PIPELINE = nil
  DEFAULT_TEMP_FILEDIR = "/tmp/"

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
  DEFAULT_EVENTS_IN_PER_MINUTE_WARNING_MIN = nil
  DEFAULT_EVENTS_IN_PER_MINUTE_WARNING_MAX = nil
  DEFAULT_EVENTS_IN_PER_MINUTE_CRITICAL_MIN = nil
  DEFAULT_EVENTS_IN_PER_MINUTE_CRITICAL_MAX = nil
  DEFAULT_EVENTS_OUT_PER_MINUTE_WARNING_MIN = nil
  DEFAULT_EVENTS_OUT_PER_MINUTE_WARNING_MAX = nil
  DEFAULT_EVENTS_OUT_PER_MINUTE_CRITICAL_MIN = nil
  DEFAULT_EVENTS_OUT_PER_MINUTE_CRITICAL_MAX = nil

  attr_accessor :host, :port, :pipeline, :temp_file_dir
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
  attr_accessor :warning_events_out_per_minute_min
  attr_accessor :warning_events_out_per_minute_max
  attr_accessor :critical_events_out_per_minute_min
  attr_accessor :critical_events_out_per_minute_max
  attr_accessor :warning_events_in_per_minute_min
  attr_accessor :warning_events_in_per_minute_max
  attr_accessor :critical_events_in_per_minute_min
  attr_accessor :critical_events_in_per_minute_max

  def initialize
    @host = DEFAULT_HOST
    @port = DEFAULT_PORT

    self.pipeline = DEFAULT_PIPELINE
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
    self.warning_events_in_per_minute_min = DEFAULT_EVENTS_IN_PER_MINUTE_WARNING_MIN
    self.warning_events_in_per_minute_max = DEFAULT_EVENTS_IN_PER_MINUTE_WARNING_MAX
    self.critical_events_in_per_minute_min = DEFAULT_EVENTS_IN_PER_MINUTE_CRITICAL_MIN
    self.critical_events_in_per_minute_max = DEFAULT_EVENTS_IN_PER_MINUTE_CRITICAL_MAX
    self.warning_events_out_per_minute_min = DEFAULT_EVENTS_OUT_PER_MINUTE_WARNING_MIN
    self.warning_events_out_per_minute_max = DEFAULT_EVENTS_OUT_PER_MINUTE_WARNING_MAX
    self.critical_events_out_per_minute_min = DEFAULT_EVENTS_OUT_PER_MINUTE_CRITICAL_MIN
    self.critical_events_out_per_minute_max = DEFAULT_EVENTS_OUT_PER_MINUTE_CRITICAL_MAX
    self.temp_file_dir = DEFAULT_TEMP_FILEDIR
  end
  
  def checks_events_in_per_minute?
    # need the saved state when events in per minute is somehow monitored
    warning_events_in_per_minute_min || warning_events_in_per_minute_max || critical_events_in_per_minute_min || critical_events_in_per_minute_max
  end

  def checks_events_out_per_minute?
    # need the saved state when events out per minute is somehow monitored
    warning_events_out_per_minute_min || warning_events_out_per_minute_max || critical_events_out_per_minute_min || critical_events_out_per_minute_max
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
    begin
      Fetcher.fetch(host, port, pipeline)
    rescue SystemExit
      exit(3)
    rescue Exception
      puts "Can not connect to Logstash"
      exit(3)
    end
  end

  def load_state
    return nil unless checks_events_in_per_minute? || checks_events_out_per_minute?

    Result.from_hash(FileHandler.read(temp_file_dir, host, port, pipeline))
  end

  def calculate_events_per_minute(state, pipeline, current_events, timestamp, direction)
    saved_events = state.get("#{pipeline}.events_#{direction}")
    saved_timestamp = state.get("#{pipeline}.timestamp")
    events_per_minute = (current_events - saved_events) / (timestamp - saved_timestamp) * 60
    events_per_minute.to_i
  end

  def performance_data(result, state)
    max_file_descriptors = result.get('process.max_file_descriptors')
    open_file_descriptors = result.get('process.open_file_descriptors')
    percent_file_descriptors = (open_file_descriptors.to_f / max_file_descriptors) * 100
    warn_file_descriptors = (max_file_descriptors / 100) * warning_file_descriptor_percent
    crit_file_descriptors = (max_file_descriptors / 100) * critical_file_descriptor_percent

    common = [
      PerfData.report_percent(result, 'process.cpu.percent', warning_cpu_percent, critical_cpu_percent, 0, 100),
      PerfData.report_percent(result, 'jvm.mem.heap_used_percent', warning_heap_percent, critical_heap_percent, 0, 100),
      PerfData.report(result, 'jvm.threads.count', nil, nil, 0, nil),
      PerfData.report(result, 'process.open_file_descriptors', warn_file_descriptors, crit_file_descriptors, 0, max_file_descriptors),
      # PerfData.report_counter(result, "pipeline.events.in", nil, nil, 0, nil),
      # PerfData.report_counter(result, "pipeline.events.filtered", nil, nil, 0, nil),
    ]

    # Inflight events per pipeline, since Logstash 6.0.0
    inflight_arr = []
    if Gem::Version.new(result.get('version')) >= Gem::Version.new('6.0.0')
      for named_pipeline in result.get('pipelines') do
        if named_pipeline[0] != ".monitoring-logstash"
          events_in = result.get('pipelines.' + named_pipeline[0] + '.events.in').to_i
          events_out = result.get('pipelines.' + named_pipeline[0] + '.events.out').to_i

          if checks_events_in_per_minute? && state.has_key?(named_pipeline[0])
            events_per_minute = calculate_events_per_minute(state, named_pipeline[0], events_in, result.get_timestamp, "in")
            inflight_arr.push(PerfData_derived.report('events_in_per_minute_' + named_pipeline[0], events_per_minute, warning_events_in_per_minute_min, warning_events_in_per_minute_max, critical_events_in_per_minute_min, critical_events_in_per_minute_max))
          end

          if checks_events_out_per_minute? && state.has_key?(named_pipeline[0])
            events_per_minute = calculate_events_per_minute(state, named_pipeline[0], events_out, result.get_timestamp, "out")
            inflight_arr.push(PerfData_derived.report('events_out_per_minute_' + named_pipeline[0], events_per_minute, warning_events_out_per_minute_min, warning_events_out_per_minute_max, critical_events_out_per_minute_min, critical_events_out_per_minute_max))
          end

          inflight_events = events_in - events_out
          inflight_arr.push(PerfData.report_counter(result, 'pipelines.' + named_pipeline[0] + '.events.in', nil, nil, 0, nil))
          inflight_arr.push(PerfData.report_counter(result, 'pipelines.' + named_pipeline[0] + '.events.out', nil, nil, 0, nil))
          inflight_arr.push(PerfData_derived.report('inflight_events_' + named_pipeline[0], inflight_events, warning_inflight_events_min, warning_inflight_events_max, critical_inflight_events_min, critical_inflight_events_max))
        end
      end
    else
      events_in = result.get('pipeline.events.in').to_i
      events_out = result.get('pipeline.events.out').to_i

      if checks_events_in_per_minute? && state.has_key?('main')
        events_per_minute = calculate_events_per_minute(state, 'main', events_in, result.get_timestamp, "in")
        inflight_arr.push(PerfData_derived.report('events_in_per_minute', events_per_minute, warning_events_in_per_minute_min, warning_events_in_per_minute_max, critical_events_in_per_minute_min, critical_events_in_per_minute_max))
      end

      if checks_events_out_per_minute? && state.has_key?('main')
        events_per_minute = calculate_events_per_minute(state, 'main', events_out, result.get_timestamp, "out")
        inflight_arr.push(PerfData_derived.report('events_out_per_minute', events_per_minute, warning_events_out_per_minute_min, warning_events_out_per_minute_max, critical_events_out_per_minute_min, critical_events_out_per_minute_max))
      end

      inflight_events = events_in - events_out
      inflight_arr.push(PerfData.report_counter(result, 'pipeline.events.in', nil, nil, 0, nil))
      inflight_arr.push(PerfData.report_counter(result, 'pipeline.events.out', nil, nil, 0, nil))
      inflight_arr.push(PerfData_derived.report('inflight_events', inflight_events, warning_inflight_events_min, warning_inflight_events_max, critical_inflight_events_min, critical_inflight_events_max))
    end

    perfdata = common + inflight_arr
    perfdata.join(' ')
  end

  # the reports are defined below, call them here

  def health(result, state)
    [
      file_descriptor_health(result),
      heap_health(result),
      inflight_events_health(result),
      config_reload_health(result),
      cpu_usage_health(result),
      events_per_minute_health(checks_events_in_per_minute?, "in", result, state, 
        warning_events_in_per_minute_min, 
        warning_events_in_per_minute_max, 
        critical_events_in_per_minute_min, 
        critical_events_in_per_minute_max),
      events_per_minute_health(checks_events_out_per_minute?, "out", result, state, 
        warning_events_out_per_minute_min, 
        warning_events_out_per_minute_max, 
        critical_events_out_per_minute_min, 
        critical_events_out_per_minute_max)
    ]
  end

  def save_events_state(result)
    return unless checks_events_in_per_minute? ||  checks_events_out_per_minute?

    stats_to_save = {}

    # Since version 6.0.0 it's possible to define multiple pipelines and give them a name.
    # This goes over all pipelines and compiles all events into one string.
    if Gem::Version.new(result.get('version')) >= Gem::Version.new('6.0.0')
      result.get('pipelines').each do |named_pipeline|
        name = named_pipeline[0]
        next if name == ".monitoring-logstash"
       
        events_in = result.get("pipelines.#{name}.events.in").to_i
        events_out = result.get("pipelines.#{name}.events.out").to_i
        stats_to_save[name] = {events_in: events_in, events_out: events_out, timestamp: result.get_timestamp}
      end
    # For versions older 6.0.0 we use the old method (unchanged)
    else
      events_in = result.get('pipeline.events.in')
      events_out = result.get('pipeline.events.out')
      stats_to_save['main'] = {events_in: events_in, events_out: events_out, timestamp: result.get_timestamp}
    end

    FileHandler.save(temp_file_dir, host, port, pipeline, stats_to_save)
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

  def events_per_minute_health(checks_event, direction, result, state, warn_min, warn_max, crit_min, crit_max)
    return unless checks_event

    # Since version 6.0.0 it's possible to define multiple pipelines and give them a name.
    # This goes over all pipelines and compiles all events into one string.
    if Gem::Version.new(result.get('version')) >= Gem::Version.new('6.0.0')
      events_report = "Events #{direction} per minute:"

      warn_counter = 0
      critical_counter = 0

      result.get('pipelines').each do |named_pipeline|
        name = named_pipeline[0]
        next if name == ".monitoring-logstash"
        next(events_report += " #{named_pipeline[0]}: Initialized;") unless state.has_key?(name)
        
        events = result.get("pipelines.#{name}.events.#{direction}").to_i
        events_per_minute = calculate_events_per_minute(state, name, events, result.get_timestamp, direction)
        events_report += " #{name}: #{events_per_minute};"

        if crit_max && crit_max < events_per_minute
          critical_counter += 1
        elsif crit_min && crit_min > events_per_minute
          critical_counter += 1
        elsif warn_max && warn_max < events_per_minute
          warn_counter += 1
        elsif warn_min && warn_min > events_per_minute
          warn_counter += 1
        end
      end

      # If any of the pipelines is above the configured values we throw the highest common alert.
      # E.g. if pipeline1 is OK, but pipeline2 is CRIT the result will be CRIT.
      if critical_counter > 0
        Critical.new(events_report)
      elsif warn_counter > 0
        Warning.new(events_report)
      else
        OK.new(events_report)
      end
    # For versions older 6.0.0 we use the old method (unchanged)
    else
      return OK.new("Events #{direction} per minute: Initialized") unless state.has_key?('main')
  
      # check if inflight events are outside of threshold
      # find a way to reuse the already computed inflight events
      events = result.get("pipeline.events.#{direction}").to_i
      events_per_minute = calculate_events_per_minute(state, 'main', events, result.get_timestamp, direction)
      events_report = "Events #{direction} per minute: #{events_per_minute}"

      if crit_max && crit_max < events_per_minute
        Critical.new(events_report)
      elsif crit_min && crit_min > events_per_minute
        Critical.new(events_report)
      elsif warn_max && warn_max < events_per_minute
        Warning.new(events_report)
      elsif warn_min && warn_min > events_per_minute
        Warning.new(events_report)
      else
        OK.new(events_report)
      end
    end
  end

  INFLIGHT_EVENTS_REPORT = 'Inflight events: %d'
  def inflight_events_health(result)
    # Since version 6.0.0 it's possible to define multiple pipelines and give them a name.
    # This goes over all pipelines and compiles all events into one string.
    if Gem::Version.new(result.get('version')) >= Gem::Version.new('6.0.0')
      inflight_events_report = 'Inflight events:'

      warn_counter = 0
      critical_counter = 0

      for named_pipeline in result.get('pipelines') do
        if named_pipeline[0] != ".monitoring-logstash"
          events_in = result.get('pipelines.' + named_pipeline[0] + '.events.in').to_i
          events_out = result.get('pipelines.' + named_pipeline[0] + '.events.out').to_i

          inflight_events = events_in - events_out
          inflight_events_report = inflight_events_report + " " + named_pipeline[0] + ": " + inflight_events.to_s + ";"

          if critical_inflight_events_max && critical_inflight_events_max < inflight_events
            critical_counter += 1
          elsif critical_inflight_events_min && critical_inflight_events_min > inflight_events
            critical_counter += 1
          elsif warning_inflight_events_max && warning_inflight_events_max < inflight_events
            warn_counter += 1
          elsif warning_inflight_events_min && warning_inflight_events_min > inflight_events
            warn_counter += 1
          end
        end
      end
      # If any of the pipelines is above the configured values we throw the highest common alert.
      # E.g. if pipeline1 is OK, but pipeline2 is CRIT the result will be CRIT.
      if critical_counter > 0
        Critical.new(inflight_events_report)
      elsif warn_counter > 0
        Warning.new(inflight_events_report)
      else
        OK.new(inflight_events_report)
      end
    # For versions older 6.0.0 we use the old method (unchanged)
    else
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
  end

  # the following would be needed to output the whole errormessage
  # CONFIG_RELOAD_REPORT = "Config reload errormessage: %s"
  CONFIG_RELOAD_REPORT = 'Config reload syntax check' 
  def config_reload_health(result)
    # Same as above: since version 6.0.0 we can have multiple pipelines.
    if Gem::Version.new(result.get('version')) >= Gem::Version.new('6.0.0')
      config_reload_errors_report = 'Config reload syntax check:'
      error_counter = 0

      for named_pipeline in result.get('pipelines') do
        if named_pipeline[0] != ".monitoring-logstash"
          error_counter += result.get('pipelines.' + named_pipeline[0] + '.reloads.failures').to_i

          config_reload_error_message = result.get('pipelines.' + named_pipeline[0] + '.reloads.last_error.message').to_s.strip
          config_reload_errors_report = config_reload_errors_report + " " + named_pipeline[0] + ": " + (config_reload_error_message.empty? ? "OK" : config_reload_error_message) + ";"
        end
      end

      if error_counter > 0
        Critical.new(config_reload_errors_report)
      else
        OK.new(config_reload_errors_report)
      end
    else
      config_reload_errors = (result.get('pipeline.reloads.failures')).to_i
      config_reload_error_message = (result.get('pipeline.reloads.last_error.message'))
      config_reload_errors_report = format(CONFIG_RELOAD_REPORT, config_reload_error_message)
      config_reload_last_success_timestamp_str = result.get('pipeline.reloads.last_success_timestamp')
      config_reload_last_failure_timestamp_str = result.get('pipeline.reloads.last_failure_timestamp')
      if config_reload_last_success_timestamp_str == nil
        config_reload_last_success_timestamp_str = '1970-01-01'
      end
      if config_reload_last_failure_timestamp_str == nil
        config_reload_last_failure_timestamp_str = '1970-01-02'
      end
      #puts config_reload_last_success_timestamp_str
      #puts config_reload_last_failure_timestamp_str
      config_reload_last_success_timestamp = DateTime.parse(config_reload_last_success_timestamp_str)
      config_reload_last_failure_timestamp = DateTime.parse(config_reload_last_failure_timestamp_str)
      #puts 'last_success_timestamp='+config_reload_last_success_timestamp.inspect
      #puts 'last_failure_timestamp='+config_reload_last_failure_timestamp.inspect
      # the following would output the whole errormessage which is too long as output of a monitoring plugin
      # config_reload_errors_report = format(CONFIG_RELOAD_REPORT, config_reload_error_message)
      if config_reload_last_success_timestamp < config_reload_last_failure_timestamp
        if config_reload_errors > 0
          Critical.new(config_reload_errors_report)
        else
          OK.new(config_reload_errors_report)
        end
      else
        OK.new(config_reload_errors_report)
      end
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
