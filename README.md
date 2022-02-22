# keylight-control

Elgato Keylight Controller for linux

![keylight-control](/screenshot.png)

## Dependencies

The Elgato Keylights are being detected by Avahi/Zeroconf multicast DNS resolution.

Depending on your Linux distribution the necessary packages my not be installed
and you have to add the `nss-mdns` package to your system. After that you have
to configure your system to use this facility for the resolution of systems in
the `/etc/nsswitch.conf`.

For Mac OS users the name resolution scheme is called Bonjour and is installed
and configured automatically.

## Usage

```
$ keylight-control
```

Or with [keylight-systray](https://github.com/mschneider82/keylight-systray)

## License

MIT

## Author

Matthias Schneider
