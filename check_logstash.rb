#!/usr/bin/env ruby
#
# File : check_logstash
# Author : Thomas Widhalm, Netways
# E-Mail: thomas.widhalm@netways.de
# Date : 01/07/2016
#
# Version: 0.2.0
#
# This program is free software; you can redistribute it or modify
# it under the terms of the GNU General Public License version 3.0
#
# Changelog:
# 	- 0.2.0 add heap and file descriptor thresholds
# 	- 0.1.0 initial, untested prototype
#
#
# Plugin check for icinga

require "rubygems"
require "json"
require "net/http"
require "uri"
require "optparse"

class CheckLogstash
  class Status
    attr_reader :message
    def initialize(message)
      @message = message
    end

    def label
      # get the class name like 'CheckLogstash::Critical' as 'CRITICAL'
      self.class.name.upcase.split("::").last
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
        "OK - Logstash looking healthy."
      end

      puts "#{status} | #{check.performance_data(result)}\n"
      puts health.sort_by(&:to_i).reverse.join("\n")

      code
    end

    def parse(check, args)
      args  = [ '-h' ] if args.empty?

      OptionParser.new do |opts|
        opts.banner = "Usage: #{$0} [options]"

        options_error = proc do |message|
          $stderr.puts message
          $stderr.puts opts
          exit 3
        end

        opts.on('-H', '--hostname HOST', 'Logstash host') { |v| check.host = v }
        opts.on('-p', '--hostname PORT', 'Logstash API port') { |v| check.port = v.to_i }
        opts.on("--file-descriptor-threshold [WARN:]CRIT", "The percentage relative to the process file descriptor limit on which to be a warning or critical result.") do |v|
          options_error.call("--file-descriptor-threshold requires an argument") if v.nil?

          values = v.split(":")
          options_error.call("--file-descriptor-threshold has invalid argument #{v}") if values.count == 0 || values.count > 2
          
          begin
            if values.count == 1
              check.warning_file_descriptor_percent = -1
              check.critical_file_descriptor_percent = values[0].to_i
            else
              check.warning_file_descriptor_percent =  values[0].to_i
              check.critical_file_descriptor_percent = values[1].to_i
            end
          rescue ArgumentError => e
            options_error.call("--file-descriptor-threshold has invalid argument. #{e.message}")
          end
        end

        opts.on("--heap-usage-threshold [WARN:]CRIT", "The percentage relative to the heap size limit on which to be a warning or critical result.") do |v|
          options_error.call("--heap-usage-threshold requires an argument") if v.nil?

          values = v.split(":")
          options_error.call("--file-descriptor-threshold has invalid argument #{v}") if values.count == 0 || values.count > 2

          begin
            if values.count == 1
              check.warning_heap_percent = nil
              check.critical_heap_percent = values[0].to_i
            else
              check.warning_heap_percent =  values[0].to_i
              check.critical_heap_percent = values[1].to_i
            end
          rescue ArgumentError => e
            options_error.call("--heap-usage-threshold has invalid argument. #{e.message}")
          end
        end
        opts.on_tail("-h", "--help", "Show this message") do
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
      first, remaining = field.split(".", 2)
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

      critical("Got HTTP response #{response.code}") if response.code != "200"

      result = begin
        JSON.parse(response.body)
      rescue => e
        critical("Failed parsing JSON response. #{e.class.name}")
      end
      Result.from_hash(result)
    end
  end

  module PerfData
    module_function
    def report(result, field, warning=nil, critical=nil, minimum=nil, maximum=nil)
      #'label'=value[UOM];[warn];[crit];[min];[max]
      format("%s=%s;%s;%s;%s;%s", field, result.get(field), warning, critical, minimum, maximum)
    end

    def report_counter(result, field, warning=nil, critical=nil, minimum=nil, maximum=nil)
      #'label'=value[UOM];[warn];[crit];[min];[max]
      # the UOM (unit of measurement) of 'c' means a counter.
      format("%s=%sc;%s;%s;%s;%s", field, result.get(field), warning, critical, minimum, maximum)
    end
  end

  DEFAULT_PORT = 9600
  DEFAULT_HOST = "127.0.0.1"

  DEFAULT_FILE_DESCRIPTOR_WARNING = 85
  DEFAULT_FILE_DESCRIPTOR_CRITICAL = 95
  DEFAULT_HEAP_WARNING = 70
  DEFAULT_HEAP_CRITICAL = 80

  attr_accessor :host, :port
  attr_accessor :warning_file_descriptor_percent
  attr_accessor :critical_file_descriptor_percent
  attr_accessor :warning_heap_percent
  attr_accessor :critical_heap_percent

  def initialize
    @host = DEFAULT_HOST
    @port = DEFAULT_PORT

    self.warning_file_descriptor_percent = DEFAULT_FILE_DESCRIPTOR_WARNING
    self.critical_file_descriptor_percent = DEFAULT_FILE_DESCRIPTOR_CRITICAL
    self.warning_heap_percent = DEFAULT_HEAP_WARNING
    self.critical_heap_percent = DEFAULT_HEAP_CRITICAL
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
    max_file_descriptors = result.get("process.max_file_descriptors")
    open_file_descriptors = result.get("process.open_file_descriptors")
    percent_file_descriptors = (open_file_descriptors.to_f / max_file_descriptors)*100

    [
      PerfData.report(result, "process.cpu.percent", nil, nil, 0, 100),
      PerfData.report(result, "mem.heap_used_percent", nil, nil, 0, 100),
      PerfData.report(result, "jvm.threads.count", nil, nil, 0, nil),
      PerfData.report_counter(result, "events.in", nil, nil, 0, nil),
      PerfData.report_counter(result, "events.filtered", nil, nil, 0, nil),
      PerfData.report_counter(result, "events.out", nil, nil, 0, nil),
    ].join(" ")
  end


  def health(result)
    [ 
      file_descriptor_health(result),
      heap_health(result),
    ]
  end

  FILE_DESCRIPTOR_REPORT = "Open file descriptors at %.2f%%. (%d out of %d file descriptors are open)"
  def file_descriptor_health(result)
    max_file_descriptors = result.get("process.max_file_descriptors")
    open_file_descriptors = result.get("process.open_file_descriptors")

    # For now, we have to compute the file descriptor usage percent until
    # Logstash stats api delivers the percentage directly.
    percent_file_descriptors = (open_file_descriptors.to_f / max_file_descriptors)*100.0

    file_descriptor_report = format(FILE_DESCRIPTOR_REPORT, percent_file_descriptors, open_file_descriptors, max_file_descriptors)

    if critical_file_descriptor_percent && percent_file_descriptors > critical_file_descriptor_percent
      Critical.new(file_descriptor_report)
    elsif warning_file_descriptor_percent && percent_file_descriptors > warning_file_descriptor_percent
      Warning.new(file_descriptor_report)
    else
      OK.new(file_descriptor_report)
    end
  end

  HEAP_REPORT = "Heap usage at %.2f%% (%d out of %d bytes in use)"
  def heap_health(result)
    percent_heap_used = result.get("mem.heap_used_percent")
    heap_report = format(HEAP_REPORT, percent_heap_used, result.get("mem.heap_used_in_bytes"), result.get("mem.heap_max_in_bytes"))

    if critical_heap_percent && percent_heap_used > critical_heap_percent
      Critical.new(heap_report)
    elsif warning_heap_percent && percent_heap_used > warning_heap_percent
      Warning.new(heap_report)
    else
      OK.new(heap_report)
    end
  end
end

if __FILE__ == $0
  exit(CheckLogstash::CLI.run(ARGV))
end
