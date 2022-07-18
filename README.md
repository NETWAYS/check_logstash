# check_logstash #

[![Rake](https://github.com/NETWAYS/check_logstash/workflows/Rake%20Tests/badge.svg)](https://github.com/NETWAYS/check_logstash/actions?query=workflow%3A%22Rake+Tests%22) [![Acceptance Tests](https://github.com/NETWAYS/check_logstash/workflows/Acceptance%20Tests/badge.svg)](https://github.com/NETWAYS/check_logstash/actions?query=workflow%3A%22Acceptance+Tests%22)

A monitoring plugin for Icinga (2), Nagios, Shinken, Naemon, etc. to check the Logstash API (Logstash v.5+)

**A word of warning** Be sure to read the configuration check part of this readme since there is a problem with this feature in past releases of Logstash.

## Usage ##

### Options ###

    -H HOST                          Logstash host
    -p, --hostname PORT              Logstash API port
    -P, --pipeline PIPELINE          Pipeline to monitor, uses all pipelines when not set
        --file-descriptor-threshold-warn WARN
                                     The percentage relative to the process file descriptor limit on which to be a warning result.
        --file-descriptor-threshold-crit CRIT
                                     The percentage relative to the process file descriptor limit on which to be a critical result.
        --heap-usage-threshold-warn WARN
                                     The percentage relative to the heap size limit on which to be a warning result.
        --heap-usage-threshold-crit CRIT
                                     The percentage relative to the heap size limit on which to be a critical result.
        --cpu-usage-threshold-warn WARN
                                     The percentage of CPU usage on which to be a warning result.
        --cpu-usage-threshold-crit CRIT
                                     The percentage of CPU usage on which to be a critical result.
        --temp-filedir NAME          Directory to use for the temporary state file. Only used when one of the events-per-minute metrics is used. Defaults to /tmp
        --inflight-events-warn WARN  Threshold for inflight events to be a warning result. Use min:max for a range.
        --inflight-events-crit CRIT  Threshold for inflight events to be a critical result. Use min:max for a range.
        --events-in-per-minute-warn WARN
                                     Threshold for the number of ingoing events per minute to be a warning. Use min:max for a range.
        --events-in-per-minute-crit CRIT
                                     Threshold for the number of ingoing events per minute to be critical. Use min:max for a range.
        --events-out-per-minute-warn WARN
                                     Threshold for the number of outgoing events per minute to be a warning. Use min:max for a range.
        --events-out-per-minute-crit CRIT
                                     Threshold for the number of outgoing events per minute to be critical. Use min:max for a range.
    -h, --help                       Show this message


### Using default values ###

    ./check_logstash.rb -H [logstashhost]
    
### Using your own thresholds ###

    ./check_logstash.rb -H 127.0.0.1 --file-descriptor-threshold-warn 40 --file-descriptor-threshold-crit 50 --heap-usage-threshold-warn 10 --heap-usage-threshold-crit 20

or

    ./check_logstash.rb -H 127.0.0.1 --inflight-events-warn 5 --inflight-events-crit 1:10

### Checking only a single pipeline
    Starting with Logstash 6.0, it is possible to use multiple pipelines. If you just want to monitor a single pipeline (e.g. You want to use independent checks for each pipleine) you can use the -P or the --pipeline parameter:

    ./check_logstash.rb -H [logstashhost] -P [pipeline_name]

### Checking events in/out per minute
    It is also possible to check the events in/out independently. This plugin uses a temporary file for this purpose, which is saved in the /tmp folder. The location can be changed using the --temp-filedir option. Make sure that the check can write to the chosen folder. The file name uses the following pattern:

    check_logstash_#{host}_#{port}_#{pipeline}_events_state.tmp

    If no specific pipeline is selected, "all" is used as the pipeline name. Note that the file is **not** created/read when no events in/out metrics are selected via the command line options. This plugin saves the current events in/out states with a timestamp in this file and on each invocation, the values are read and the current events in/out per minute metrics are calculated. Afterwards, the new state is saved in the file.

    The first invocation of this plugin with events in/out monitoring initiates the temporary file, so the corresponding metrics are only shown on the next invocation.

## Sample Output ##

### With default values ###

    OK - Logstash looking healthy. | process.cpu.percent=0;;;0;100 mem.heap_used_percent=18;70;80;0;100 jvm.threads.count=23;;;0; process.open_file_descriptors=46;3400;3800;0;4096 pipeline.events.out=166c;;;0; inflight_events=0;;;0;
    OK: Inflight events: 0
    OK: Heap usage at 18.00% (382710568 out of 2077753344 bytes in use)
    OK: Open file descriptors at 1.12%. (46 out of 4096 file descriptors are open)

### With thresholds set ###

    CRITICAL - Logstash is unhealthy - CRITICAL: Inflight events: 0 | process.cpu.percent=0;;;0;100 mem.heap_used_percent=16;70;80;0;100 jvm.threads.count=23;;;0; process.open_file_descriptors=46;3400;3800;0;4096 pipeline.events.out=164c;;;0; inflight_events=0;5;10;0;
    CRITICAL: Inflight events: 0
    OK: Heap usage at 16.00% (352959904 out of 2077753344 bytes in use)
    OK: Open file descriptors at 1.12%. (46 out of 4096 file descriptors are open)

### With events in/out per minute set ###

    OK - Logstash seems to be doing fine. | process.cpu.percent=0%;;;0;100 jvm.mem.heap_used_percent=46%;70;80;0;100 jvm.threads.count=38;;;0; process.open_file_descriptors=128;891225;996075;0;1048576 events_in_per_minute_main=2070;1:;1: events_out_per_minute_main=2069;1:;1: pipelines.main.events.in=236178654c;;;0; pipelines.main.events.out=236178650c;;;0; inflight_events_main=4;;
    OK: Events out per minute: main: 2069;
    OK: Events in per minute: main: 2070;
    OK: CPU usage in percent: 0
    OK: Config reload syntax check: main: OK;
    OK: Inflight events: main: 4;
    OK: Heap usage at 46.00% (486260792 out of 1038876672 bytes in use)
    OK: Open file descriptors at 0.01%. (128 out of 1048576 file descriptors are open)

## With events in/out per minute set and with two pipelines ###

    CRITICAL - Logstash is unhealthy - CRITICAL: Events in per minute: PipelineOne: 2497; PipelineTwo: 0; | process.cpu.percent=11%;;;0;100 jvm.mem.heap_used_percent=70%;70;80;0;100 jvm.threads.count=592;;;0; process.open_file_descriptors=526;3400;3800;0;4096 events_in_per_minute_PipelineOne=2497;1:;1: events_out_per_minute_PipelineOne=2479;1:;1: pipelines.PipelineOne.events.in=23289504c;;;0; pipelines.PipelineOne.events.out=23289493c;;;0; inflight_events_PipelineOne=11;; events_in_per_minute_PipelineTwo=0;1:;1: events_out_per_minute_PipelineTwo=0;1:;1: pipelines.PipelineTwo.events.in=6606c;;;0; pipelines.PipelineTwo.events.out=6606c;;;0; inflight_events_PipelineTwo=0;; 
    CRITICAL: Events out per minute: PipelineOne: 2479; PipelineTwo: 0;
    CRITICAL: Events in per minute: PipelineOne: 2497; PipelineTwo: 0; 
    OK: CPU usage in percent: 11
    OK: Config reload syntax check: PipelineOne: OK; PipelineTwo: 
    OK: Heap usage at 70.00% (736537928 out of 1037959168 bytes in use)
    OK: Open file descriptors at 12.84%. (526 out of 4096 file descriptors are open)

## Finding viable thresholds ##

To set your thresholds for inflight events to a sensible value use baselining. Don't set thresholds from the beginning but let Graphite or other graphers create graphs for inflight events. Just add some percent to what Logstash usually processes and set this as threshold. Or use the `generator` plugin to put as many events through your Elastic stack as possible. Use some percent (e.g. 90%) from this maximum as a threshold. Keep in mind that changing your configuration might change the maximum inflight events.

### Configuration check ###

Logstash 5.0 can automatically reload changed configuration from disk. This plugin checks if the last reload succeeded or failed. Unfortunately the first release of Logstash 5.0 does not provide a way to show that it recovered from an invalid configuration. If you had an error, the plugin will still show that there is a problem with the configuration even when this is already fixed. There is already an issue with Logstash pending to fix this behaviour: https://github.com/elastic/logstash/issues/6149

## Default values ##

There are some default values defined in the plugin. Some values are merely put out as performance data and some are normally used for performance data but can be checked against thresholds.

### Checks with defaults ###

* `-H`: 127.0.0.1
* `-p`: 9600
* `--file-descriptor-threshold-warn` : 85
* `--file-descriptor-threshold-warn` : 95
* `--heap-usage-threshold-warn` : 70
* `--heap-usage-threshold-warn` : 80

### Optionally checked ###

* `--pipeline`
* `--cpu-usage-threshold-warn`
* `--cpu-usage-threshold-crit`
* `--inflight-events-warn`
* `--inflight-events-crit`
* `--events-in-per-minute-warn`
* `--events-in-per-minute-crit`
* `--events-out-per-minute-warn`
* `--events-out-per-minute-crit`

## Building ##

While `check_logstash` is the finished plugin you can use, you might want to change something and rebuild the script.

Simply issue the `rake` command (if you have Rake installed) and it will run tests (and rubocop when all cops are pleased) and create the script from the library files. *Don't change the `check_logstash` file since it will be overwritten by Rake, change the files in `lib/` instead.
