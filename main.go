package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

// ServerEntry holds per-server CyberArk connection details.
// Can be specified as a simple string (vault name) or a full mapping.
type ServerEntry struct {
	Vault   string `yaml:"vault"`
	Account string `yaml:"account"` // target/privileged account (e.g. root, admin)
	Domain  string `yaml:"domain"`  // optional domain for the target account
}

// UnmarshalYAML allows servers to be specified as either a simple string (vault)
// or a full object with vault/account/domain fields.
func (s *ServerEntry) UnmarshalYAML(value *yaml.Node) error {
	// Try simple string first (backwards compatible).
	if value.Kind == yaml.ScalarNode {
		s.Vault = value.Value
		return nil
	}
	// Otherwise unmarshal as struct.
	type raw ServerEntry
	return value.Decode((*raw)(s))
}

// Config represents the ~/.cyberark-ssh.yaml configuration file.
type Config struct {
	User           string                  `yaml:"user"`
	CyberArkHost   string                  `yaml:"cyberark_host"`
	Port           int                     `yaml:"port"` // PSMP SSH port (default 22)
	DefaultVault   string                  `yaml:"default_vault"`
	DefaultAccount string                  `yaml:"default_account"` // default target account
	DefaultDomain  string                  `yaml:"default_domain"`  // default domain
	Servers        map[string]*ServerEntry `yaml:"servers"`         // server/alias -> entry
	Aliases        map[string]string       `yaml:"aliases"`         // short name -> full hostname
	SSHArgs        []string                `yaml:"ssh_args"`        // extra args passed to ssh
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".cyberark-ssh.yaml")
}

func loadConfig() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config %s: %w\nRun '%s init' to create a sample config", path, err, os.Args[0])
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config %s: %w", path, err)
	}

	if cfg.User == "" {
		return nil, fmt.Errorf("config: 'user' is required")
	}
	if cfg.CyberArkHost == "" {
		return nil, fmt.Errorf("config: 'cyberark_host' is required")
	}

	return &cfg, nil
}

// resolveAlias expands a short alias to its full hostname.
// Returns the original name if no alias is found.
func (cfg *Config) resolveAlias(name string) string {
	if full, ok := cfg.Aliases[name]; ok {
		return full
	}
	return name
}

// lookupServer finds the ServerEntry for a given server hostname.
// Falls back to defaults if the server isn't explicitly configured.
func (cfg *Config) lookupServer(server string) (*ServerEntry, bool) {
	if entry, ok := cfg.Servers[server]; ok {
		return entry, true
	}
	return nil, false
}

// effectiveEntry returns a ServerEntry with all defaults filled in.
func (cfg *Config) effectiveEntry(server string) (*ServerEntry, error) {
	entry, found := cfg.lookupServer(server)

	if !found {
		if cfg.DefaultVault == "" {
			return nil, fmt.Errorf("server %q not found in config and no default_vault set", server)
		}
		fmt.Fprintf(os.Stderr, "note: server %q not in config, using defaults\n", server)
		entry = &ServerEntry{}
	}

	// Fill in defaults for any empty fields.
	if entry.Vault == "" {
		entry.Vault = cfg.DefaultVault
	}
	if entry.Account == "" {
		entry.Account = cfg.DefaultAccount
	}
	if entry.Domain == "" {
		entry.Domain = cfg.DefaultDomain
	}

	if entry.Vault == "" {
		return nil, fmt.Errorf("no vault configured for server %q and no default_vault set", server)
	}

	return entry, nil
}

// buildConnStr constructs the CyberArk PSMP connection string.
//
// Formats supported by CyberArk PSMP:
//
//	user@vault@target@PSMPhost                         (basic)
//	user@account@target@PSMPhost                       (with target account)
//	user@account#domain@target@PSMPhost                (with domain)
func buildConnStr(user string, entry *ServerEntry, server, cyberarkHost string) string {
	// Build the middle segment: vault or account[#domain]
	middle := entry.Vault
	if entry.Account != "" {
		middle = entry.Account
		if entry.Domain != "" {
			middle += "#" + entry.Domain
		}
	}

	return fmt.Sprintf("%s@%s@%s@%s", user, middle, server, cyberarkHost)
}

func writeExampleConfig() error {
	path := configPath()
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s — remove it first if you want to regenerate", path)
	}

	example := `# CyberArk SSH wrapper configuration
# =====================================
#
# user:            your CyberArk user ID
# cyberark_host:   the CyberArk PSM/PSMP proxy host
# port:            SSH port for the PSMP host (default: 22)
# default_vault:   vault/safe used when a server has no explicit mapping
# default_account: default target/privileged account (e.g. root, admin)
# default_domain:  default domain for target accounts
# ssh_args:        extra arguments passed to every ssh invocation
# aliases:         short names that expand to full hostnames
# servers:         mapping of target server -> CyberArk vault/account details
#
# Connection string format:
#   ssh [-p port] user@account#domain@target@cyberark_host
#   ssh [-p port] user@vault@target@cyberark_host
#
# Servers can be specified as a simple string (vault name) or a full object:
#   simple:  server_name: vault_name
#   full:    server_name:
#              vault: vault_name
#              account: root           # target/privileged account
#              domain: mydomain.com    # optional domain

user: twa7331
cyberark_host: capsmp.work.sfgcorp.com
# port: 22

default_vault: kube_test
# default_account: root
# default_domain: ""

# Common SSH args for CyberArk PSMP (host keys can rotate):
ssh_args:
  - "-o"
  - "StrictHostKeyChecking=no"
  - "-o"
  - "UserKnownHostsFile=/dev/null"

# Short aliases for long hostnames
aliases:
  mgr1: vsr-t-k8sc1mgr1
  mgr2: vsr-t-k8sc1mgr2
  # db: vsr-p-db01

# Server-to-vault mappings
# Simple form (just vault name):
servers:
  vsr-t-k8sc1mgr1: kube_test
  vsr-t-k8sc1mgr2: kube_test

  # Full form (vault + target account + domain):
  # vsr-p-app01:
  #   vault: prod_vault
  #   account: root
  #   domain: prod.corp.com
`
	if err := os.WriteFile(path, []byte(example), 0600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}
	fmt.Printf("wrote example config to %s — edit it with your servers and vaults\n", path)
	return nil
}

func printUsage() {
	bin := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `CyberArk SSH wrapper — simplify SSH through CyberArk PSMP

Usage:
  %[1]s <server|alias>               SSH to a server using its mapped vault
  %[1]s <server|alias> [ssh args]    SSH with extra arguments forwarded to ssh
  %[1]s scp <src> <dst>              SCP through CyberArk (use :<server>:path)
  %[1]s list                         List configured servers, aliases, and vaults
  %[1]s show <server|alias>          Show the SSH command that would be run
  %[1]s init                         Write example config to ~/.cyberark-ssh.yaml
  %[1]s help                         Show this help

Connection string format:
  ssh user@vault@target@cyberark_host
  ssh user@account#domain@target@cyberark_host

Config file: ~/.cyberark-ssh.yaml
`, bin)
}

func listServers(cfg *Config) {
	// Print aliases if any.
	if len(cfg.Aliases) > 0 {
		aliases := make([]string, 0, len(cfg.Aliases))
		for a := range cfg.Aliases {
			aliases = append(aliases, a)
		}
		sort.Strings(aliases)

		fmt.Printf("%-20s %s\n", "ALIAS", "EXPANDS TO")
		fmt.Printf("%-20s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 40))
		for _, a := range aliases {
			fmt.Printf("%-20s %s\n", a, cfg.Aliases[a])
		}
		fmt.Println()
	}

	// Print servers.
	if len(cfg.Servers) == 0 {
		fmt.Println("no servers configured — edit", configPath())
	} else {
		servers := make([]string, 0, len(cfg.Servers))
		for s := range cfg.Servers {
			servers = append(servers, s)
		}
		sort.Strings(servers)

		fmt.Printf("%-40s %-20s %-15s %s\n", "SERVER", "VAULT", "ACCOUNT", "DOMAIN")
		fmt.Printf("%-40s %-20s %-15s %s\n",
			strings.Repeat("-", 40), strings.Repeat("-", 20),
			strings.Repeat("-", 15), strings.Repeat("-", 15))
		for _, s := range servers {
			e := cfg.Servers[s]
			vault := e.Vault
			if vault == "" {
				vault = cfg.DefaultVault + " (default)"
			}
			account := e.Account
			if account == "" && cfg.DefaultAccount != "" {
				account = cfg.DefaultAccount + " (default)"
			}
			if account == "" {
				account = "-"
			}
			domain := e.Domain
			if domain == "" && cfg.DefaultDomain != "" {
				domain = cfg.DefaultDomain + " (default)"
			}
			if domain == "" {
				domain = "-"
			}
			fmt.Printf("%-40s %-20s %-15s %s\n", s, vault, account, domain)
		}
	}

	fmt.Printf("\ndefault vault:   %s\n", valueOrNone(cfg.DefaultVault))
	fmt.Printf("default account: %s\n", valueOrNone(cfg.DefaultAccount))
	fmt.Printf("default domain:  %s\n", valueOrNone(cfg.DefaultDomain))
	fmt.Printf("cyberark host:   %s\n", cfg.CyberArkHost)
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	fmt.Printf("port:            %d\n", port)
	fmt.Printf("user:            %s\n", cfg.User)
	if len(cfg.SSHArgs) > 0 {
		fmt.Printf("ssh_args:        %s\n", strings.Join(cfg.SSHArgs, " "))
	}
}

func valueOrNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// doSSH builds and executes the SSH command for a given server.
func doSSH(cfg *Config, server string, extraArgs []string, dryRun bool) {
	// Resolve alias.
	server = cfg.resolveAlias(server)

	entry, err := cfg.effectiveEntry(server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	connStr := buildConnStr(cfg.User, entry, server, cfg.CyberArkHost)

	// Assemble full ssh command.
	sshArgs := []string{"ssh"}

	// Add port if non-default.
	port := cfg.Port
	if port != 0 && port != 22 {
		sshArgs = append(sshArgs, "-p", strconv.Itoa(port))
	}

	sshArgs = append(sshArgs, cfg.SSHArgs...)
	sshArgs = append(sshArgs, extraArgs...)
	sshArgs = append(sshArgs, connStr)

	fmt.Fprintf(os.Stderr, "→ %s\n", strings.Join(sshArgs, " "))

	if dryRun {
		return
	}

	// exec into ssh so it fully replaces this process (proper TTY handling).
	sshBin, err2 := exec.LookPath("ssh")
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "error: ssh not found in PATH: %v\n", err2)
		os.Exit(1)
	}

	if err2 := syscall.Exec(sshBin, sshArgs, os.Environ()); err2 != nil {
		fmt.Fprintf(os.Stderr, "error: exec ssh: %v\n", err2)
		os.Exit(1)
	}
}

// doSCP handles scp through CyberArk. Remote paths use the format :<server>:path
// Example: cyberark-ssh scp localfile.txt :mgr1:/tmp/remotefile.txt
func doSCP(cfg *Config, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "error: scp requires source and destination\n")
		fmt.Fprintf(os.Stderr, "usage: %s scp <src> <dst>\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  remote paths use :<server|alias>:<path> format\n")
		fmt.Fprintf(os.Stderr, "  example: %s scp file.txt :mgr1:/tmp/file.txt\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	scpArgs := []string{"scp"}

	// Add port if non-default.
	port := cfg.Port
	if port != 0 && port != 22 {
		scpArgs = append(scpArgs, "-P", strconv.Itoa(port))
	}

	scpArgs = append(scpArgs, cfg.SSHArgs...)

	// Process remaining args, expanding :<server>:<path> to CyberArk format.
	for _, arg := range args {
		if strings.HasPrefix(arg, ":") {
			// Remote path: :<server>:<path>
			rest := arg[1:]
			parts := strings.SplitN(rest, ":", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "error: invalid remote path %q — use :<server>:<path>\n", arg)
				os.Exit(1)
			}
			server := cfg.resolveAlias(parts[0])
			remotePath := parts[1]

			entry, err := cfg.effectiveEntry(server)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}

			connStr := buildConnStr(cfg.User, entry, server, cfg.CyberArkHost)
			scpArgs = append(scpArgs, connStr+":"+remotePath)
		} else {
			scpArgs = append(scpArgs, arg)
		}
	}

	fmt.Fprintf(os.Stderr, "→ %s\n", strings.Join(scpArgs, " "))

	scpBin, err := exec.LookPath("scp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: scp not found in PATH: %v\n", err)
		os.Exit(1)
	}

	if err := syscall.Exec(scpBin, scpArgs, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "error: exec scp: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "help", "-h", "--help":
		printUsage()
		return

	case "init":
		if err := writeExampleConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return

	case "list", "ls":
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		listServers(cfg)
		return

	case "show":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: %s show <server|alias>\n", filepath.Base(os.Args[0]))
			os.Exit(1)
		}
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		doSSH(cfg, os.Args[2], os.Args[3:], true)
		return

	case "scp":
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		doSCP(cfg, os.Args[2:])
		return
	}

	// Default: treat first arg as the target server.
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	doSSH(cfg, cmd, os.Args[2:], false)
}
