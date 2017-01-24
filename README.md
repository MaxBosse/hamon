# HaMon - Haproxy Monitor

Goal of this tool is to have a simple small monitor for multiple loadbalancers.

Example output:
```
HaMon 0.0.3.2 - HEAD HEAD 2016-07-03_01:46:16
Global Sessions: 2220           Global SessionsRate: 1090

Haproxy 1        Sessions: 330   SessionRate: 597
└── stage-www            stage-www  Server has no check defined!

Haproxy 2    Sessions: 1188  SessionRate: 198

Haproxy 3        Sessions: 196   SessionRate: 146
├── default		        www0       DOWN for 25s
└── default		        www1       DOWN for 1m2s


Last update: 2016-07-03 03:46:58
```