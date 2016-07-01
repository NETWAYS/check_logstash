# check_logstash #
A monitoring plugin for Icinga (2), Nagios, Shinken, Naemon, etc. to check the Logstash API (Logstash v.5+)

This is still under heavy development and needs testing.

## Usage ##

    ./check_logstash.rb -h [logstashhost]
    
More options, like thresholds, etc, are to come.

## Sample Output ##

    Ok: Logstash is doing fine In-flight events: 0| cpu_percent=1%;80;90;0;100; memory_usage=3199213568;;;;; events_in=3055c;;;;; events_filtered=3055c;;;;; events_out=3055c;;;;; events_pending_filter=0; events_pending_output=0; events_in_flight=0;10;20; open_file_descriptors=71;80;90;;4096;
