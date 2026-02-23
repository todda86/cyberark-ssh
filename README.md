# cyberark-ssh

A Go CLI wrapper that simplifies SSH and SCP connections through CyberArk PSMP (Privileged Session Manager Proxy).

Instead of typing:

```
ssh twa7331@kube_test@vsr-t-k8sc1mgr1@capsmp.work.sfgcorp.com
```

Just type:

```
cyberark-ssh mgr1
```

## How It Works

CyberArk PSMP requires an SSH connection string in the format:

```
user@vault@target_server@cyberark_host
user@account#domain@target_server@cyberark_host
```

`cyberark-ssh` reads a YAML config file (`~/.cyberark-ssh.yaml`) that maps target servers to their CyberArk vaults, accounts, and domains, then constructs and executes the full SSH command for you. It uses `syscall.Exec` to replace itself with the `ssh`/`scp` process, so TTY and interactive sessions work exactly as if you ran `ssh` directly.

## Installation

### From source

Requires [Go 1.21+](https://go.dev/dl/).

```bash
git clone <repo-url>
cd cyberark-ssh
go build -o cyberark-ssh .
```

Copy the binary somewhere on your PATH:

```bash
cp cyberark-ssh /usr/local/bin/
```

Or install directly via Go:

```bash
go install .
```

## Quick Start

1. **Generate a starter config file:**

   ```bash
   cyberark-ssh init
   ```

   This creates `~/.cyberark-ssh.yaml` with example values.

2. **Edit the config** with your actual user ID, CyberArk host, and server-to-vault mappings:

   ```bash
   $EDITOR ~/.cyberark-ssh.yaml
   ```

3. **Connect to a server:**

   ```bash
   cyberark-ssh vsr-t-k8sc1mgr1
   # or using an alias:
   cyberark-ssh mgr1
   ```

## Usage

```
cyberark-ssh <server|alias>               SSH to a server using its mapped vault
cyberark-ssh <server|alias> [ssh args]    SSH with extra arguments forwarded to ssh
cyberark-ssh scp <src> <dst>              SCP through CyberArk (use :<server>:path)
cyberark-ssh list                         List configured servers, aliases, and vaults
cyberark-ssh show <server|alias>          Show the SSH command that would be run (dry run)
cyberark-ssh init                         Write example config to ~/.cyberark-ssh.yaml
cyberark-ssh help                         Show help
```

### SSH Examples

```bash
# Basic connection
cyberark-ssh vsr-t-k8sc1mgr1

# Using an alias
cyberark-ssh mgr1

# Port forwarding
cyberark-ssh mgr1 -L 8080:localhost:80

# Verbose SSH output
cyberark-ssh mgr1 -v

# Run a remote command
cyberark-ssh mgr1 -- uptime

# Preview the command without connecting
cyberark-ssh show mgr1
```

### SCP Examples

Remote paths use the `:<server|alias>:<path>` format:

```bash
# Copy a local file to a remote server
cyberark-ssh scp localfile.txt :mgr1:/tmp/remotefile.txt

# Copy a remote file to local
cyberark-ssh scp :mgr1:/var/log/app.log ./app.log

# Copy between two remote servers
cyberark-ssh scp :mgr1:/tmp/data.tar :mgr2:/tmp/data.tar
```

### List and Inspect

```bash
# List all configured servers, aliases, and vaults
cyberark-ssh list

# Show the exact SSH command that would be executed (dry run)
cyberark-ssh show vsr-t-k8sc1mgr1
```

When you run a command, the tool prints the constructed SSH command to stderr before executing:

```
→ ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null twa7331@kube_test@vsr-t-k8sc1mgr1@capsmp.work.sfgcorp.com
```

## Configuration

The config file lives at `~/.cyberark-ssh.yaml`. Run `cyberark-ssh init` to generate a starter file.

### Full Example

```yaml
# Your CyberArk user ID
user: twa7331

# CyberArk PSM/PSMP proxy host
cyberark_host: capsmp.work.sfgcorp.com

# SSH port for the PSMP host (default: 22, omit if standard)
# port: 2222

# Vault to use when a server isn't explicitly mapped
default_vault: kube_test

# Default target/privileged account (e.g. root, admin)
# default_account: root

# Default domain for target accounts
# default_domain: corp.com

# Extra arguments passed to every ssh/scp invocation.
# Recommended: disable host key checking since PSMP host keys can rotate.
ssh_args:
  - "-o"
  - "StrictHostKeyChecking=no"
  - "-o"
  - "UserKnownHostsFile=/dev/null"

# Short aliases for long hostnames — type "mgr1" instead of the full name
aliases:
  mgr1: vsr-t-k8sc1mgr1
  mgr2: vsr-t-k8sc1mgr2
  wrk1: vsr-t-k8sc1wrk1
  db:   vsr-p-db01

# Server-to-vault mappings
servers:
  # Simple form: just the vault name
  vsr-t-k8sc1mgr1: kube_test
  vsr-t-k8sc1mgr2: kube_test
  vsr-t-k8sc1wrk1: kube_test

  # Full form: vault + target account + domain
  vsr-p-app01:
    vault: prod_vault
    account: root
    domain: prod.corp.com

  vsr-p-db01:
    vault: prod_db_vault
    account: oracle
```

### Server Entry Formats

Servers can be specified in two ways:

**Simple** — just a vault name (string):
```yaml
servers:
  my-server: my_vault
```

**Full** — an object with vault, account, and domain:
```yaml
servers:
  my-server:
    vault: my_vault
    account: root           # target/privileged account on the remote host
    domain: prod.corp.com   # optional domain for the target account
```

Both formats can be mixed in the same config.

### Config Fields

| Field | Required | Default | Description |
|---|---|---|---|
| `user` | **Yes** | — | Your CyberArk user ID |
| `cyberark_host` | **Yes** | — | CyberArk PSMP hostname |
| `port` | No | `22` | SSH port for the PSMP host |
| `default_vault` | No | — | Vault used for servers not in `servers` map |
| `default_account` | No | — | Default target/privileged account (e.g. `root`) |
| `default_domain` | No | — | Default domain for target accounts |
| `ssh_args` | No | — | Extra arguments passed to every `ssh`/`scp` call |
| `aliases` | No | — | Map of `short_name: full_hostname` |
| `servers` | No | — | Map of `hostname: vault_or_entry` |

### Connection String Formats

The tool builds one of these connection strings depending on what's configured:

| Config present | Connection string |
|---|---|
| vault only | `user@vault@server@psmp_host` |
| account (no domain) | `user@account@server@psmp_host` |
| account + domain | `user@account#domain@server@psmp_host` |

When an `account` is set on a server entry, it takes precedence over `vault` as the middle segment of the connection string (this matches CyberArk PSMP's expected format).

### Default Vault Behavior

If you connect to a server that isn't explicitly listed under `servers`:

- **With `default_vault` set:** The default vault is used and a note is printed to stderr.
- **Without `default_vault`:** The command exits with an error.

This means you only need to map servers that use a *different* vault from your default. If all your servers use the same vault, just set `default_vault` and leave `servers` empty.

### Aliases

Aliases let you type short names instead of full hostnames:

```yaml
aliases:
  mgr1: vsr-t-k8sc1mgr1
  db:   vsr-p-db01
```

Then `cyberark-ssh mgr1` expands to the full hostname before looking up vault/account.

### Recommended SSH Args

CyberArk PSMP host keys can change when the proxy rotates or is redeployed. To avoid `known_hosts` conflicts:

```yaml
ssh_args:
  - "-o"
  - "StrictHostKeyChecking=no"
  - "-o"
  - "UserKnownHostsFile=/dev/null"
```

### Non-Standard Ports

If your PSMP host listens on a port other than 22:

```yaml
port: 2222
```

The tool automatically adds `-p 2222` (for ssh) or `-P 2222` (for scp).

## File Permissions

The config file is created with `0600` permissions (readable only by you) since it contains your user ID and infrastructure details.

## Troubleshooting

**"cannot read config" error**
Run `cyberark-ssh init` to create the config file, then edit it.

**"config already exists" on init**
The config file already exists. Remove it first if you want to regenerate:
```bash
rm ~/.cyberark-ssh.yaml
cyberark-ssh init
```

**"server not found in config and no default_vault set"**
Either add the server to the `servers` map or set `default_vault` in the config.

**Preview the command without connecting**
Use `show` to see exactly what would be executed:
```bash
cyberark-ssh show myserver
```

**SSH connection issues**
Use the `-v` flag to get verbose SSH output:
```bash
cyberark-ssh myserver -v
```
The tool prints the exact SSH command it runs (prefixed with `→`) so you can compare it with a working manual command.

**Host key verification failures**
Add these to `ssh_args` in your config (PSMP host keys can rotate):
```yaml
ssh_args:
  - "-o"
  - "StrictHostKeyChecking=no"
  - "-o"
  - "UserKnownHostsFile=/dev/null"
```
