# Cirri Daemon

## 1. enable *.host.ona.im DNS for portable cirri dev

Because most people don't have static IP's, and the VPN isn't always up - or maybe they're on a plane.

- run a local DNS server that makes and uses a local loopback alias to permit localhost style connection without using loopback
- watch to see if Docker daemon is running (Docker Desktop on demand...)
- could do the relaying to external hosts?
- auto configure host to use it
  - OSX: https://passingcuriosity.com/2013/dnsmasq-dev-osx/
  - Linux: use systemd-resolved options

## 2. start a desktop systray app when the user logs in..

cos we can do fun UX then - like changing the DOCKERSOCKET to a remote host - or access slurm, d2iq or whatever

## 3. improve the local host volume mapping experience

especially for remote cluster

## Non-functional work

- install to /usr/local/bin/cirrid-VERSION and use softlink to /usr/local/bin/cirrid
- regular check for updates to help users keep up to date
- tracing to enable debugging
