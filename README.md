# mackerel-plugin-linux-process-status

mackerel metric plugin for CPU/FDs/Memory usage of each linux process.

## Usage

```
$ ./mackerel-plugin-linux-process-status -h
Usage:
  mackerel-plugin-linux-process-status [OPTIONS]

Application Options:
  -p, --pid=        PID
      --key-prefix= Metric key prefix
  -v, --version     Show version

Help Options:
  -h, --help        Show this help message
```

Sample

```
$ ./mackerel-plugin-linux-process-status --key-prefix test -p 54321
process-status.fds_test.count   6       1606457504
process-status.fds_test.max     65535   1606457504
process-status.fds_usage_test.percentage        0.009155        1606457504
process-status.cpu_test.percentage      0.000000        1606457504
process-status.mem_test.used    2924544 1606457504
process-status.mem_test.max     469286912       1606457504
process-status.mem_usage_test.percentage        0.623189        1606457504
```

## Limitation

This plugin does not monitor I/O wait time or CPU steal time.

## Install

Please download release page or `mkr plugin install monitoring-forge/mackerel-plugin-linux-process-status`.