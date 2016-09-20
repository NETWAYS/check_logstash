# check_logstash #
A monitoring plugin for Icinga (2), Nagios, Shinken, Naemon, etc. to check the Logstash API (Logstash v.5+)

This is still under heavy development and needs testing.

## Usage ##

### Using default values ###

    ./check_logstash.rb -H [logstashhost]
    
### Using your own thresholds ###

    ./check_logstash.rb -H 127.0.0.1 --file-descriptor-threshold-warn 40 --file-descriptor-threshold-crit 50 --heap-usage-threshold-warn 10 --heap-usage-threshold-crit 20

## Sample Output ##

### With default values ###

OK - Logstash looking healthy. | process.cpu.percent=0;;;0;100 mem.heap_used_percent=17;70;80;0;100 jvm.threads.count=23;;;0; process.open_file_descriptors=46;3400;3800;0;4096 pipeline.events.in=238c;;;0; pipeline.events.filtered=238c;;;0; pipeline.events.out=238c;;;0;
OK: Heap usage at 17.00% (359577784 out of 2077753344 bytes in use)
OK: Open file descriptors at 1.12%. (46 out of 4096 file descriptors are open)

### With thresholds set ###

WARNING - Logstash may not be healthy - WARNING: Heap usage at 17.00% (373565816 out of 2077753344 bytes in use) | process.cpu.percent=0;;;0;100 mem.heap_used_percent=17;10;20;0;100 jvm.threads.count=23;;;0; process.open_file_descriptors=46;1600;2000;0;4096 pipeline.events.in=234c;;;0; pipeline.events.filtered=234c;;;0; pipeline.events.out=234c;;;0;
WARNING: Heap usage at 17.00% (373565816 out of 2077753344 bytes in use)
OK: Open file descriptors at 1.12%. (46 out of 4096 file descriptors are open)
