# Backing up your account

When you first run `charm`, it creates a new ED25519 key pair for you. That
private key is the __key__ to your data.

To back it up, you can use the `backup-keys` command, as such:

```shell 
charm backup-keys 
```

It'll create a `charm-keys-backup.tar` file in the current folder. You can
override the path by passing a `-o` flag, as such:

```shell 
charm backup-keys -o ~/charm.tar 
```

You may also print the private key to STDOUT in order to pipe it into other
command, such as [`melt`](https://github.com/charmbracelet/melt). Example
usage:

```shell 
charm backup-keys -o - | melt 
```

Also worth reading [./docs/restore-account.md](./restore-account.md).
