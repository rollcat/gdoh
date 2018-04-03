# gdoh

A.k.a. Go DNS-over-HTTPS

## What?

- <https://1.1.1.1/>
- <https://developers.cloudflare.com/1.1.1.1/dns-over-https/>

## How?

Get it:

    go get github.com/rollcat/gdoh

Try it:

    gdoh -listen :1253

In another terminal:

    dig @127.0.0.1 -p 1253 rollc.at +short

Use it! Run it listening on `:53` (the default), either as root or
with `CAP_NET_BIND_SERVICE` (see [`capabilities(7)`][capabilities.7]).

(Sadly, root privileges can't be dropped after binding the socket -
see [Go issue #1435][go-1435].)

Put `nameserver 127.0.0.1` in your `/etc/resolv.conf` or equivalent.

[capabilities.7]: https://linux.die.net/man/7/capabilities
[go-1435]: https://github.com/golang/go/issues/1435
