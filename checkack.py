#!/usr/bin/python
import sys

args=sys.argv[1:]
if len(args) < 2:
    print 'args: ackNo bitmap...'
    sys.exit(1)

on=int(args[0])

args=''.join(args[1:])
args=args.replace('[', '')
args=args.replace(']', '')
print args

def getBits():
	global args
	bits=list(args[:16])
	args=args[16:]
	for i in range(8):
		bits[i]=bits[14-i]
		bits[i+1]=bits[15-i]
	return int(''.join(bits), 16)

while len(args)>0:
	bits=getBits()
	for i in range(64):
		print '%d %s' % (on, '.' if bits&1 else 'miss')
		bits>>=1
		on+=1

