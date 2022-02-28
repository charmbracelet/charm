# Running Charm with Docker

The official Charm images are available at [charmcli/charm](https://hub.docker.com/r/charmcli/charm). Development and nightly builds are available at [ghcr.io/charmbracelet/charm](https://ghcr.io/charmbracelet/charm).

```sh
docker pull charmcli/charm:latest
```

Here’s how you might run `charm` as a container. Keep in mind that
the database is stored in the `/data` directory, so you’ll likely want
to mount that directory as a volume in order keep your your data backed up.

```sh
docker run \
  --name=charm \
  -v /path/to/data:/data \
  -p 35353:35353 \
  -p 35354:35354 \
  -p 35355:35355 \
  -p 35356:35356 \
  --restart unless-stopped \
  charmcli/charm:latest
```

or by using `docker-compose`:

```yaml
version: "3.1"
services:
  charm:
    image: charmcli/charm:latest
    container_name: charm
    volumes:
      - /path/to/data:/data
    ports:
      - 35353:35353
      - 35354:35354
      - 35355:35355
      - 35356:35356
    restart: unless-stopped
```

To set up TLS under Docker, consider using a reverse proxy such as
[traefik](https://doc.traefik.io/traefik/https/overview/) or a web server with
automatic HTTPS like [caddy](https://caddyserver.com/docs/automatic-https). If
you're using a reverse proxy, you will need to set `CHARM_SERVER_HOST` to your
public host, and `CHARM_SERVER_PUBLIC_URL` to the full public URL of your
reverse proxy i.e.  `CHARM_SERVER_PUBLIC_URL=https://cloud.charm.sh:35354`.

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="the Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source


[releases]: https://github.com/charmbracelet/charm/releases
[docs]: https://pkg.go.dev/github.com/charmbracelet/charm?tab=doc
[kv]: https://github.com/charmbracelet/charm/tree/main/kv
[fs]: https://github.com/charmbracelet/charm/tree/main/fs
[crypt]: https://github.com/charmbracelet/charm/tree/main/crypt
[glow]: https://github.com/charmbracelet/glow
[skate]: https://github.com/charmbracelet/skate
[badger]: https://github.com/dgraph-io/badger
