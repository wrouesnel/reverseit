[![Build](https://github.com/wrouesnel/reverseit/actions/workflows/release.yml/badge.svg)](https://github.com/wrouesnel/reverseit/actions/workflows/release.yml)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/reverseit/badge.svg?branch=master)](https://coveralls.io/github/wrouesnel/reverseit?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/wrouesnel/reverseit)](https://goreportcard.com/report/github.com/wrouesnel/reverseit)


# reverseit

Utility to implement steerable reverse SSH tunnel servers.

## Motivation
For some reason I could not find anything like this anywhere else on the net. 
This utility allows you to have SSH clients connect to a bastion host, and then have a port on their side forwarded 
to a service on the connecting machine.

The goal is similar to a tool like [frp](https://github.com/fatedier/frp) or [rathole](https://github.com/rapiz1/rathole)
but is designed to just be slotting into your `.ssh/authorized_keys` file to grant options if encountering an unexpected
scenario.

## Usage

### Server Side:
Use `reverseit server` in your authorized keys file to specify which port to
listen on for connections back to the client.

Example:
```bash
# ~/.ssh/authorized_keys
command="reverseit server :2201" <ssh key here>
```

Connecting to this host with the key you put as the SSH key will open a local port of :2201 which
forwards connections back over the link to the reverseit client instance.

### Client Side:
The client should SSH to the server with the correct key. stdin and stdout
are linked to the `reverseit client` process.

Call `reverseit client` with an executable where stdin/stdout will land on a `reverseit server`
instance (typically `ssh` but any anything which works with stdin/stdout will do).

```bash
reverseit client 127.0.0.1:22 -- ssh -T <server>
```

It's recommended to use a systemd service with restart policy to make this persistent. 
See the [example unit file](reverseit.systemd).

## Testing Locally

To test the reverseit will work for you, it's generally possible to just run it locally in one command line.
The following works provided you have passwordless loopback SSH (`ssh localhost` logs you on to your own machine):

Before doing this ensure you have built a binary for your system with `make reverseit`.

```bash
$(pwd)/reverseit --log-level=debug client 127.0.0.1:22 -- ssh -T localhost $(pwd)/reverseit --log-level=debug server :2201
```

Then check it's working in another shell:

```bash
ssh -p 2201 localhost
```