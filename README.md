# Cirri Daemon

## installation

```
# download latest release
chmod 755 ./cirrid
sudo ./cirrid install
```

On our internal OSX boxes, you'll need to become ading first - GUI, or `ComputerAdminCLI --add`.

To see the serice log output:

* Linux: `sudo journalctl -fu cirrid`
* OSX: `cat /usr/local/var/log/cirrid.*`
* Windows: sorry, not implemented yet

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

## TODO:

2. use goreleaser
3. prom metrics endpoint
4. tui to edit / add / remove entries
5. a cirri container watcher that looks at the autosave.json and auto adds dns entries (with user able to cfg on/off) 
6. seriously debug why there's a hickup in resolving dns - and ~20 dns requests per lookup? (this may be only the first time after flushing the cache..)
7. `cirrid status` - tell me what dns values are set, some metrics?
8. `cirrid logs` - tail the logs?
9. watch the inifile, reload...
10. ini setting for log level
