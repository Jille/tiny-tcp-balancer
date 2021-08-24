# Tiny TCP balancer

A small Go program that receives connections on `--port` and forwards them to the first of the given `--targets` that accepted a TCP connection.

Can be used to make a simple and small HA proxy.
