# check_logstash

An Icinga check plugin to check Logstash.

## Usage

```bash
Usage:
  check_logstash [flags]
  check_logstash [command]

Available Commands:
  health      Checks the health of the Logstash server
  pipeline    Checks the status of the Logstash Pipelines

Flags:
  -H, --hostname string    Hostname of the Logstash server (default "localhost")
  -p, --port int           Port of the Logstash server (default 9600)
  -s, --secure             Use a HTTPS connection
  -i, --insecure           Skip the verification of the server's TLS certificate
  -b, --bearer string      Specify the Bearer Token for server authentication
  -u, --user string        Specify the user name and password for server authentication <user:password>
      --ca-file string     Specify the CA File for TLS authentication
      --cert-file string   Specify the Certificate File for TLS authentication
      --key-file string    Specify the Key File for TLS authentication
  -t, --timeout int        Timeout in seconds for the CheckPlugin (default 30)
  -h, --help               help for check_logstash
  -v, --version            version for check_logstash
```

### Health

Checks the health status of the Logstash server.

```bash
Usage:
  check_logstash health [flags]

Examples:

	$ check_logstash health --hostname 'localhost' --port 8888 --insecure
	OK - Logstash is healthy | status=green process.cpu.percent=0;0.5;3;0;100
	 \_[OK] Heap usage at 12.00%
	 \_[OK] Open file descriptors at 12.00%
	 \_[OK] CPU usage at 5.00%

	$ check_logstash -p 9600 health --cpu-usage-threshold-warn 50 --cpu-usage-threshold-crit 75
	WARNING - CPU usage at 55.00%
	 \_[OK] Heap usage at 12.00%
	 \_[OK] Open file descriptors at 12.00%
	 \_[WARNING] CPU usage at 55.00%

Flags:
      --file-descriptor-threshold-warn string   The percentage relative to the process file descriptor limit on which to be a warning result (default "100")
      --file-descriptor-threshold-crit string   The percentage relative to the process file descriptor limit on which to be a critical result (default "100")
      --heap-usage-threshold-warn string        The percentage relative to the heap size limit on which to be a warning result (default "70")
      --heap-usage-threshold-crit string        The percentage relative to the heap size limit on which to be a critical result (default "80")
      --cpu-usage-threshold-warn string         The percentage of CPU usage on which to be a warning result (default "100")
      --cpu-usage-threshold-crit string         The percentage of CPU usage on which to be a critical result (default "100")
  -h, --help                                    help for health
```

### Pipeline

Determines the health of Logstash pipelines via "inflight events". These events are calculated as such: `inflight events = events.In - events.Out`

Hint: Use the queue backpressure for Logstash 8.

```bash
Usage:
  check_logstash pipeline [flags]

Examples:

	$ check_logstash pipeline --inflight-events-warn 5 --inflight-events-crit 10
	WARNING - Inflight events
	 \_[WARNING] inflight_events_example-input:9;
	 \_[OK] inflight_events_example-default-connector:4

	$ check_logstash pipeline --inflight-events-warn 5 --inflight-events-crit 10 --pipeline example
	CRITICAL - Inflight events
	 \_[CRITICAL] inflight_events_example:15

Flags:
  -P, --pipeline string               Pipeline Name (default "/")
      --inflight-events-warn string   Warning threshold for inflight events to be a warning result. Use min:max for a range.
      --inflight-events-crit string   Critical threshold for inflight events to be a critical result. Use min:max for a range.
  -h, --help                          help for pipeline
```

### Pipeline Flow Metrics

Checks the status of a Logstash pipeline's flow metrics (currently queue backpressure).

Hint: Requires Logstash 8.5.0

```bash

Usage:
  check_logstash pipeline flow [flags]

Examples:

	$ check_logstash pipeline flow --warning 5 --critical 10
	OK - Flow metrics alright
	 \_[OK] queue_backpressure_example:0.34;

	$ check_logstash pipeline flow --pipeline example --warning 5 --critical 10
	CRITICAL - Flow metrics alright
	 \_[CRITICAL] queue_backpressure_example:11.23;

Flags:
  -c, --critical string   Critical threshold for queue Backpressure
  -h, --help              help for flow
  -P, --pipeline string   Pipeline Name (default "/")
  -w, --warning string    Warning threshold for queue Backpressure
```

### Pipeline Reload

Checks the status of Logstash pipelines configuration reload.

```bash
Usage:
  check_logstash pipeline reload [flags]

Examples:

	$ check_logstash pipeline reload
	OK - Configuration successfully reloaded
	 \_[OK] Configuration successfully reloaded for pipeline Foobar for on 2021-01-01T02:07:14Z

	$ check_logstash pipeline reload --pipeline Example
	CRITICAL - Configuration reload failed
	 \_[CRITICAL] Configuration reload for pipeline Example failed on 2021-01-01T02:07:14Z

Flags:
  -P, --pipeline string               Pipeline Name (default "/")
  -h, --help                          help for pipeline
```

## License

Copyright (c) 2022 [NETWAYS GmbH](mailto:info@netways.de)

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public
License as published by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not,
see [gnu.org/licenses](https://www.gnu.org/licenses/).
