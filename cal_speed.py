#!/usr/bin/python
import sys

def speed(latency, win):
	return (1000.0/latency)*win*1400.0/1024.0

if len(sys.argv) == 3:
	latency = int(sys.argv[1])
	win = int(sys.argv[2])
	print "Theoretical speed %.2fKB/s" % speed(latency, win)
else:
	print "%s [latency] [win]" % sys.argv[0]

