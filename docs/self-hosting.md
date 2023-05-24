# Self-Hosting Charm

Charm libraries point at our Charmbracelet, Inc. servers by default (that’s
*cloud.charm.sh*), however it's very easy for users to host their own Charm
instances. The charm binary is a single, statically-linked executable capable
of serving an entire Charm instance. 

## Ze Server

To start your charm server, run `charm serve` in a dedicated terminal window or
in a [Docker container](https://github.com/charmbracelet/charm/blob/main/docker.md). 
Then, change the default host by adding `CHARM_HOST=localhost` or
`CHARM_HOST=burrito.example.com` to your PATH. 

## Ze Client

If you're using a reverse proxy with your self-hosted Charm server, you'll want
to change a few environment variables. Namely,

* `CHARM_HOST`: This should match the public URL to your Charm server.
* `CHARM_HTTP_PORT`: This should match the port your reverse proxy accepts for HTTP connections.
* `CHARM_SERVER_PUBLIC_URL`: This is the public URL set on your Charm server. 

By default, the `CHARM_HTTP_PORT` value is set to `35354`. If you're using a
default HTTP reverse proxy, you'll need to change the reverse proxy to accept
port `35354` for HTTP connections or change the `CHARM_HTTP_PORT` to `443` on
the client side. 

## Self-Hosting With TLS

### About our Setup

We're hosting our infrastructure on AWS. The Charm instance uses 2 load
balancers, one is layer 4 (NLB) for handling SSH requests, and the other is
layer 7 (ALB) for handling HTTPS SSL/TLS requests. TLS gets terminated at the
load balancer level, then the ALB communicates with the Charm instance in plain
HTTP no-TLS.

The NLB handles incoming traffic using a TCP listener on port `35353` and
forwards that to the Charm instance port `35353`. The ALB handles incoming
traffic using an HTTPS listener on port `35354`, terminates TLS, and forwards
plain HTTP to the Charm instance on port `35354`

### Using Your Own TLS Certificate

If you want to use your own TLS certificate, you could specify
`CHARM_SERVER_USE_TLS`, `CHARM_SERVER_TLS_KEY_FILE`, and
`CHARM_SERVER_TLS_CERT_FILE`. In this case, the Charm HTTP server will handle
TLS terminations.

### Configuring Your VPS

In nginx, you could set up Let's Encrypt, SSL termination, and HTTPS/SSL on
port `35354`, then use proxy_pass to reverse proxy the requests to your Charm
instance. For SSH port `35353`, you'd just need to make sure that this port
accepts incoming traffic on the VPS.

Helpful resources:  
[1] https://docs.nginx.com/nginx/admin-guide/security-controls/terminating-ssl-http/  
[2] https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/  
[3] https://upcloud.com/community/tutorials/install-lets-encrypt-nginx/  

## Storage Restrictions

The self-hosting max data is disabled by default. You can change that using
`CHARM_SERVER_USER_MAX_STORAGE`
