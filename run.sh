#!/bin/bash

sysctl -w fs.file-max=100000
ulimit -n  99999
/root/port-scanner/portscanner -log-file test.log 
