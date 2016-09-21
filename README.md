# check_logstash #
A monitoring plugin for Icinga (2), Nagios, Shinken, Naemon, etc. to check the Logstash API (Logstash v.5+)

This is still under heavy development and needs testing.

## Usage ##

### Options ###

    -H HOST                          Logstash host
    -p, --hostname PORT              Logstash API port
        --file-descriptor-threshold-warn WARN
                                     The percentage relative to the process file descriptor limit on which to be a warning result.
        --file-descriptor-threshold-crit CRIT
                                     The percentage relative to the process file descriptor limit on which to be a critical result.
        --heap-usage-threshold-warn WARN
                                     The percentage relative to the heap size limit on which to be a warning result.
        --heap-usage-threshold-crit CRIT
                                     The percentage relative to the heap size limit on which to be a critical result.
        --inflight-events-warn WARN  Threshold for inflight events to be a warning result. Use min:max for a range.
        --inflight-events-crit CRIT  Threshold for inflight events to be a critical result. Use min:max for a range.
    -h, --help                       Show this message


### Using default values ###

    ./check_logstash.rb -H [logstashhost]
    
### Using your own thresholds ###

    ./check_logstash.rb -H 127.0.0.1 --file-descriptor-threshold-warn 40 --file-descriptor-threshold-crit 50 --heap-usage-threshold-warn 10 --heap-usage-threshold-crit 20

or

    ./check_logstash.rb -H 127.0.0.1 --inflight-events-warn 5 --inflight-events-crit 1:10

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

## Finding viable thresholds ##

To set your thresholds for inflight events to a sensnible value use baselining. Don't set thresholds from the beginning but let Graphite or other graphers create graphs for inflight events. Just add some percent to what Logstash usually processes and set this as threshold. Or use the `generator` plugin to put as many events through your Elastic stack as possible. Use some percent (e.g. 90%) from this maximum as a threshold. Keep in mind that changing your configuration might change the maximum inflight events.
