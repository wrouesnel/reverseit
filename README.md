[![Build Status](https://travis-ci.org/wrouesnel/reverseit.svg?branch=master)](https://travis-ci.org/wrouesnel/reverseit)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/reverseit/badge.svg?branch=master)](https://coveralls.io/github/wrouesnel/reverseit?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/wrouesnel/reverseit)](https://goreportcard.com/report/github.com/wrouesnel/reverseit)


# reverseit

Utility to implement steerable reverse SSH tunnel servers.

## Motivation
For some reason I could not find anything like this anywhere else on the
net. This utility allows you to have SSH clients connect to a bastion host,
and then have a port on their side forwarded to a port you control on your side.

## Usage

### Server Side:
Use `reverseit server` in your authorized keys file to specify which port to
listen on and forward back to the client.

`authorized_keys`
```
command="reverseit server 2201" <ssh key here>
```

### Client Side:
Connect with SSH and forward stdin and stdout to `reverseit`:

```
mkfifo inputpipe
ssh <server> < inputpipe | reverseit 22 > inputpipe 
```

For convenience `reverseit` will also handle this on its own:
```
reverseit -c 'ssh <server>' 22
```

For super-convenience it will also restart the command when it exits:
```
reverseit -c 'ssh <server>' -forever 22
```