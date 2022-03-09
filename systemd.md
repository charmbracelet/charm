# Running Charm with Systemd

Running `charm` as a systemd service is fairly straightforward. Create a file
called `/etc/systemd/system/charm.service`:

```config
[Unit]
Description=The mystical Charm Cloud 🌟
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=1
Environment=CHARM_SERVER_DATA_DIR=/var/lib/charm
ExecStart=/usr/bin/charm serve

[Install]
WantedBy=multi-user.target
```

* Set the proper `charm` binary path in `ExecStart=`
* Set where the data should be stored at in `CHARM_SERVER_DATA_DIR`

If you’re using TLS, don’t forget to set the appropriate environment variables
in the systemd service file as described below.

## TLS

See [TLS](README.md#tls) for more information.

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="the Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source

