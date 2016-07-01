#!/usr/bin/ruby
#
# File : check_logstash
# Author : Thomas Widhalm, Netways
# E-Mail: thomas.widhalm@netways.de
# Date : 01/07/2016
#
# Version: 0.1.0
#
# This program is free software; you can redistribute it or modify
# it under the terms of the GNU General Public License version 3.0
#
# Changelog:
# 	- 0.1.0 initial, untested prototype
#
#
# Plugin check for icinga

require "rubygems"
require "json"
require "net/http"
require "uri"
require "optparse"

## default values

# missing: in flight events

# percent of open file decriptors

file_descriptors_warn = 80
file_descriptors_crit = 90

# percent of cpu usage

cpu_warn = 80
cpu_crit = 90

# total memory in bytes (default max heap of Logstash is 1 GB)
# verify if -XmX java setting and this value are referencing the same data

#mem_warn = (1024**3)*0.8
#mem_crit = (1024**3)*0.9

# inflight events (total input - total output)
# find a better way to determine thresholds

inflight_warn = 10
inflight_crit = 20

warnstatus = false
critstatus = false
critstring = ""
warnstring = ""

## read options
# reference: http://ruby-doc.org/stdlib-1.9.3/libdoc/optparse/rdoc/OptionParser.html
# reference: http://stackoverflow.com/questions/4244611/pass-variables-to-ruby-script-via-command-line

options = {}

OptionParser.new do |opts|
    opts.banner = "Usage: check_logstash [options]"
    
    opts.on('-H', '--hostname HOST', 'Logstash host') { |v| options[:lshost] = v }
    opts.on('-p', '--hostname PORT', 'Logstash API port') { |v| options[:lsport] = v }
    opts.on_tail("-h", "--help", "Show this message") do
      puts opts
      exit
    end
end.parse!

#puts options

## set default values

if options[:lsport].nil?
    options[:lsport] = 9600
end


## read from API

uri = URI.parse("http://#{options[:lshost]}:#{options[:lsport]}/_node/stats")

http = Net::HTTP.new(uri.host, uri.port)
request = Net::HTTP::Get.new(uri.request_uri)

response = http.request(request)

if response.code == "200"
  result = JSON.parse(response.body)
  
#  result.each do |doc|
#    #puts doc["id"] #reference properties like this
#    puts doc # this is the result in object form    
#    puts ""
#    puts ""
#  end
# print "Events: #{result['events']['in']}"
  
  pre_filter = result['events']['in'] - result['events']['filtered']
  filtered = result['events']['filtered'] - result['events']['out']
  inflight = result['events']['in'] - result['events']['out']
  
  if result['process']['cpu']['percent'] > cpu_crit
      critstatus = true
      critstring = critstring + " CPU: #{result['process']['cpu']['percent']}%"
  elsif result['process']['cpu']['percent'] > cpu_warn
      warnstatus = true
      warnstring = warnstring + " CPU: #{result['process']['cpu']['percent']}%"
  end
  
  if inflight > inflight_crit
      critstatus = true
      # inflight events are printed no matter the status
  elsif inflight > inflight_warn
      warnstatus = true
  end
  
  if result['process']['open_file_descriptors'] > file_descriptors_crit
      critstatus = true
      critstring = critstring + " open file descriptors: #{result['process']['open_file_descriptors']}"
  elsif result['process']['open_file_descriptors'] > file_descriptors_warn
      warnstatus = true
      warnstring = warnstring + " open file descriptors: #{result['process']['open_file_descriptors']}"
  end
  
  if critstatus
      print "CRITICAL: " + critstring
  elsif warnstatus
      print "WARNING: " + warnstring
  else
      print "OK: Logstash is doing fine"
  end
  
  print " In-flight events: #{inflight}" 
  
  print "| "
  print "cpu_percent=#{result['process']['cpu']['percent']}%;#{cpu_warn};#{cpu_crit};0;100;"
  print " "
  print "memory_usage=#{result['process']['mem']['total_virtual_in_bytes']};;;;;"
  print " "
  print "events_in=#{result['events']['in']}c;;;;;"
  print " "
  print "events_filtered=#{result['events']['filtered']}c;;;;;"
  print " "
  print "events_out=#{result['events']['out']}c;;;;;"
  print " "
  print "events_pending_filter=#{pre_filter};"
  print " "
  print "events_pending_output=#{filtered};"
  print " "
  print "events_in_flight=#{inflight};#{inflight_warn};#{inflight_crit};"
  print " "
  print "open_file_descriptors=#{result['process']['open_file_descriptors']};#{file_descriptors_warn};#{file_descriptors_crit};;#{result['process']['max_file_descriptors']};"
  puts " "
  if critstatus
      exit 2
  elsif warnstatus
      exit 1
  else
      exit 0
  end
else
  puts "Logstash API not reachable"
  exit 3
end
