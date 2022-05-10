# Restoring from a backup

To restore your account, you can use the `import-keys` command:

```shell
charm import-keys charm-keys-backup.tar
```

You can also import a private key from STDIN from another tool, such as [melt](https://github.com/charmbracelet/melt):

```shell
cat seed.txt | melt restore - | charm import-keys
```

Also worth reading [./docs/backup-account.md](./backup-account.md).
