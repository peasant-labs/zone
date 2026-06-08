# Zone

Zone builds and runs Dockerized workspaces for agent harnesses.

## Local Binary Alias

If you have built or checked out the local binary at `bin/zone`, you can make the `zone` command point at it with a shell alias.

From the repository root:

```sh
alias zone="$PWD/bin/zone"
```

Verify it works:

```sh
zone --help
```

## Persist The Alias

Add the alias to your shell startup file so new terminals can use it.

For Bash:

```sh
echo 'alias zone="/absolute/path/to/zone/bin/zone"' >> ~/.bashrc
source ~/.bashrc
```

For Zsh:

```sh
echo 'alias zone="/absolute/path/to/zone/bin/zone"' >> ~/.zshrc
source ~/.zshrc
```

Replace `/absolute/path/to/zone` with this repository's absolute path. From the repository root, you can print that path with:

```sh
pwd
```

## Alternative: Add `bin` To PATH

If you prefer not to use an alias, add the repo's `bin` directory to `PATH`.

Bash:

```sh
echo 'export PATH="/absolute/path/to/zone/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Zsh:

```sh
echo 'export PATH="/absolute/path/to/zone/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```
