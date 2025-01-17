#!/bin/bash

echo 0 > /proc/sys/net/ipv4/conf/eth0/send_redirects

/usr/bin/supervisord
