# Charm

Making the command line friendly!

## Usage

Library docs to come. For now build `cmd/charm`.

### Picking a name

You can pick a name for your account by running `charm name NAME`. If the name is already taken, just run it again with a different, cooler name.

### Linking machines and keys

Charm makes it easy to link multiple machines or keys with your Charm account. To start the process run `charm link` on a machine that already has your account setup. Then on another machine enter `charm link CODE` but use the code given to you in the first command. Once confirmed, your accounts should be nice and linked up.

### JWT

JWT tokens are a way to authenticate to different web services that utilize your Charm account. If you're a nerd you can `charm jwt` to get one for yourself.

### ID

Want to know your Charm ID? You're in luck! Just `charm id` to receive your answer.

### Keys

Charm accounts are powered by SSH keys. You can see all of the keys linked to your account with `charm keys`.
