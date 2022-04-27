# Bringing your own keys

The	`charm` CLI will always create a new public/private key of ed25519 keys for you.

However, you can also link and use your existing keys to your account.

To link a key to an account, run:

```sh
charm keys -a my-key.pub
```

Or, if you want to add the keys from your SSH agent:

```sh
charm keys -a $(ssh-add -L)
```

Once you've done that, you can use your keys and SSH agent:

```sh
CHARM_IDENTITY_KEY=my-key charm keys
```

Or, if you want to use the keys from your SSH agent:

```sh
CHARM_USE_SSH_AGENT=true charm keys
```

You can put these settings in your `~/.bashrc` or similar to make them persistent.
