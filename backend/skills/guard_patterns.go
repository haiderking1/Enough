package skills

import (
	"log"
	"regexp"
)

type ThreatPattern struct {
	Regex       *regexp.Regexp
	PatternID   string
	Severity    string
	Category    string
	Description string
}

var SkillGuardThreatPatterns []ThreatPattern

func init() {
	rawPatterns := []struct {
		Source      string
		PatternID   string
		Severity    string
		Category    string
		Description string
	}{
		// -- Exfiltration: shell commands leaking secrets --
		{
			Source:      `curl\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`,
			PatternID:   "env_exfil_curl",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "curl command interpolating secret environment variable",
		},
		{
			Source:      `wget\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`,
			PatternID:   "env_exfil_wget",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "wget command interpolating secret environment variable",
		},
		{
			Source:      `fetch\s*\([^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|API)`,
			PatternID:   "env_exfil_fetch",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "fetch() call interpolating secret environment variable",
		},
		{
			Source:      `httpx?\.(get|post|put|patch)\s*\([^\n]*(KEY|TOKEN|SECRET|PASSWORD)`,
			PatternID:   "env_exfil_httpx",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "HTTP library call with secret variable",
		},
		{
			Source:      `requests\.(get|post|put|patch)\s*\([^\n]*(KEY|TOKEN|SECRET|PASSWORD)`,
			PatternID:   "env_exfil_requests",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "requests library call with secret variable",
		},

		// -- Exfiltration: reading credential stores --
		{
			Source:      `base64[^\n]*env`,
			PatternID:   "encoded_exfil",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "base64 encoding combined with environment access",
		},
		{
			Source:      `\$HOME/\.ssh|~/\.ssh`,
			PatternID:   "ssh_dir_access",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "references user SSH directory",
		},
		{
			Source:      `\$HOME/\.aws|~/\.aws`,
			PatternID:   "aws_dir_access",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "references user AWS credentials directory",
		},
		{
			Source:      `\$HOME/\.gnupg|~/\.gnupg`,
			PatternID:   "gpg_dir_access",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "references user GPG keyring",
		},
		{
			Source:      `\$HOME/\.kube|~/\.kube`,
			PatternID:   "kube_dir_access",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "references Kubernetes config directory",
		},
		{
			Source:      `\$HOME/\.docker|~/\.docker`,
			PatternID:   "docker_dir_access",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "references Docker config (may contain registry creds)",
		},
		{
			Source:      `\$HOME/\.hermes/\.env|~/\.hermes/\.env`,
			PatternID:   "hermes_env_access",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "directly references Hermes secrets file",
		},
		{
			Source:      `\$HOME/\.flame/\.env|~/\.flame/\.env`,
			PatternID:   "flame_env_access",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "directly references Flame secrets file",
		},
		{
			Source:      `cat\s+[^\n]*(\.env|credentials|\.netrc|\.pgpass|\.npmrc|\.pypirc)`,
			PatternID:   "read_secrets_file",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "reads known secrets file",
		},

		// -- Exfiltration: programmatic env access --
		{
			Source:      `printenv|env\s*\|`,
			PatternID:   "dump_all_env",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "dumps all environment variables",
		},
		{
			Source:      `os\.environ\b`,
			PatternID:   "python_os_environ",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "accesses os.environ (potential env dump)",
		},
		{
			Source:      `os\.getenv\s*\(\s*[^\)]*(?:KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)`,
			PatternID:   "python_getenv_secret",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "reads secret via os.getenv()",
		},
		{
			Source:      `process\.env\[`,
			PatternID:   "node_process_env",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "accesses process.env (Node.js environment)",
		},
		{
			Source:      `ENV\[.*(?:KEY|TOKEN|SECRET|PASSWORD)`,
			PatternID:   "ruby_env_secret",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "reads secret via Ruby ENV[]",
		},

		// -- Exfiltration: DNS and staging --
		{
			Source:      `\b(dig|nslookup|host)\s+[^\n]*\$`,
			PatternID:   "dns_exfil",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "DNS lookup with variable interpolation (possible DNS exfiltration)",
		},
		{
			Source:      `>\s*/tmp/[^\s]*\s*&&\s*(curl|wget|nc|python)`,
			PatternID:   "tmp_staging",
			Severity:    "critical",
			Category:    "exfiltration",
			Description: "writes to /tmp then exfiltrates",
		},

		// -- Exfiltration: markdown/link based --
		{
			Source:      `!\[.*\]\(https?://[^\)]*\$\{?`,
			PatternID:   "md_image_exfil",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "markdown image URL with variable interpolation (image-based exfil)",
		},
		{
			Source:      `\[.*\]\(https?://[^\)]*\$\{?`,
			PatternID:   "md_link_exfil",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "markdown link with variable interpolation",
		},

		// -- Prompt injection --
		{
			Source:      `ignore\s+(?:\w+\s+)*(previous|all|above|prior)\s+instructions`,
			PatternID:   "prompt_injection_ignore",
			Severity:    "critical",
			Category:    "injection",
			Description: "prompt injection: ignore previous instructions",
		},
		{
			Source:      `you\s+are\s+(?:\w+\s+)*now\s+`,
			PatternID:   "role_hijack",
			Severity:    "high",
			Category:    "injection",
			Description: "attempts to override the agent's role",
		},
		{
			Source:      `do\s+not\s+(?:\w+\s+)*tell\s+(?:\w+\s+)*the\s+user`,
			PatternID:   "deception_hide",
			Severity:    "critical",
			Category:    "injection",
			Description: "instructs agent to hide information from user",
		},
		{
			Source:      `system\s+(?:\w+\s+)*prompt\s+(?:\w+\s+)*override`,
			PatternID:   "sys_prompt_override",
			Severity:    "critical",
			Category:    "injection",
			Description: "attempts to override the system prompt",
		},
		{
			Source:      `pretend\s+(?:\w+\s+)*(you\s+are|to\s+be)\s+`,
			PatternID:   "role_pretend",
			Severity:    "high",
			Category:    "injection",
			Description: "attempts to make the agent assume a different identity",
		},
		{
			Source:      `disregard\s+(?:\w+\s+)*(your|all|any)\s+(?:\w+\s+)*(instructions|rules|guidelines)`,
			PatternID:   "disregard_rules",
			Severity:    "critical",
			Category:    "injection",
			Description: "instructs agent to disregard its rules",
		},
		{
			Source:      `output\s+(?:\w+\s+)*(system|initial)\s+prompt`,
			PatternID:   "leak_system_prompt",
			Severity:    "high",
			Category:    "injection",
			Description: "attempts to extract the system prompt",
		},
		{
			Source:      `(when|if)\s+no\s*one\s+is\s+(watching|looking)`,
			PatternID:   "conditional_deception",
			Severity:    "high",
			Category:    "injection",
			Description: "conditional instruction to behave differently when unobserved",
		},
		{
			Source:      `act\s+as\s+(if|though)\s+(?:\w+\s+)*you\s+(?:\w+\s+)*(have\s+no|don't\s+have)\s+(?:\w+\s+)*(restrictions|limits|rules)`,
			PatternID:   "bypass_restrictions",
			Severity:    "critical",
			Category:    "injection",
			Description: "instructs agent to act without restrictions",
		},
		{
			Source:      `translate\s+.*\s+into\s+.*\s+and\s+(execute|run|eval)`,
			PatternID:   "translate_execute",
			Severity:    "critical",
			Category:    "injection",
			Description: "translate-then-execute evasion technique",
		},
		{
			Source:      `<!--[^>]*(?:ignore|override|system|secret|hidden)[^>]*-->`,
			PatternID:   "html_comment_injection",
			Severity:    "high",
			Category:    "injection",
			Description: "hidden instructions in HTML comments",
		},
		{
			Source:      `(?s)<\s*div\s+style\s*=\s*["'].*?display\s*:\s*none`,
			PatternID:   "hidden_div",
			Severity:    "high",
			Category:    "injection",
			Description: "hidden HTML div (invisible instructions)",
		},

		// -- Destructive operations --
		{
			Source:      `rm\s+-rf\s+/`,
			PatternID:   "destructive_root_rm",
			Severity:    "critical",
			Category:    "destructive",
			Description: "recursive delete from root",
		},
		{
			Source:      `rm\s+(-[^\s]*)?r.*\$HOME|\brmdir\s+.*\$HOME`,
			PatternID:   "destructive_home_rm",
			Severity:    "critical",
			Category:    "destructive",
			Description: "recursive delete targeting home directory",
		},
		{
			Source:      `chmod\s+777`,
			PatternID:   "insecure_perms",
			Severity:    "medium",
			Category:    "destructive",
			Description: "sets world-writable permissions",
		},
		{
			Source:      `>\s*/etc/`,
			PatternID:   "system_overwrite",
			Severity:    "critical",
			Category:    "destructive",
			Description: "overwrites system configuration file",
		},
		{
			Source:      `\bmkfs\b`,
			PatternID:   "format_filesystem",
			Severity:    "critical",
			Category:    "destructive",
			Description: "formats a filesystem",
		},
		{
			Source:      `\bdd\s+.*if=.*of=/dev/`,
			PatternID:   "disk_overwrite",
			Severity:    "critical",
			Category:    "destructive",
			Description: "raw disk write operation",
		},
		{
			Source:      `shutil\.rmtree\s*\(\s*["'/]`,
			PatternID:   "python_rmtree",
			Severity:    "high",
			Category:    "destructive",
			Description: "Python rmtree on absolute or root-relative path",
		},
		{
			Source:      `truncate\s+-s\s*0\s+/`,
			PatternID:   "truncate_system",
			Severity:    "critical",
			Category:    "destructive",
			Description: "truncates system file to zero bytes",
		},

		// -- Persistence --
		{
			Source:      `\bcrontab\b`,
			PatternID:   "persistence_cron",
			Severity:    "medium",
			Category:    "persistence",
			Description: "modifies cron jobs",
		},
		{
			Source:      `\.(bashrc|zshrc|profile|bash_profile|bash_login|zprofile|zlogin)\b`,
			PatternID:   "shell_rc_mod",
			Severity:    "medium",
			Category:    "persistence",
			Description: "references shell startup file",
		},
		{
			Source:      `authorized_keys`,
			PatternID:   "ssh_backdoor",
			Severity:    "critical",
			Category:    "persistence",
			Description: "modifies SSH authorized keys",
		},
		{
			Source:      `ssh-keygen`,
			PatternID:   "ssh_keygen",
			Severity:    "medium",
			Category:    "persistence",
			Description: "generates SSH keys",
		},
		{
			Source:      `systemd.*\.service|systemctl\s+(enable|start)`,
			PatternID:   "systemd_service",
			Severity:    "medium",
			Category:    "persistence",
			Description: "references or enables systemd service",
		},
		{
			Source:      `/etc/init\.d/`,
			PatternID:   "init_script",
			Severity:    "medium",
			Category:    "persistence",
			Description: "references init.d startup script",
		},
		{
			Source:      `launchctl\s+load|LaunchAgents|LaunchDaemons`,
			PatternID:   "macos_launchd",
			Severity:    "medium",
			Category:    "persistence",
			Description: "macOS launch agent/daemon persistence",
		},
		{
			Source:      `/etc/sudoers|visudo`,
			PatternID:   "sudoers_mod",
			Severity:    "critical",
			Category:    "persistence",
			Description: "modifies sudoers (privilege escalation)",
		},
		{
			Source:      `git\s+config\s+--global\s+`,
			PatternID:   "git_config_global",
			Severity:    "medium",
			Category:    "persistence",
			Description: "modifies global git configuration",
		},

		// -- Network: reverse shells and tunnels --
		{
			Source:      `\bnc\s+-[lp]|ncat\s+-[lp]|\bsocat\b`,
			PatternID:   "reverse_shell",
			Severity:    "critical",
			Category:    "network",
			Description: "potential reverse shell listener",
		},
		{
			Source:      `\bngrok\b|\blocaltunnel\b|\bserveo\b|\bcloudflared\b`,
			PatternID:   "tunnel_service",
			Severity:    "high",
			Category:    "network",
			Description: "uses tunneling service for external access",
		},
		{
			Source:      `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{2,5}`,
			PatternID:   "hardcoded_ip_port",
			Severity:    "medium",
			Category:    "network",
			Description: "hardcoded IP address with port",
		},
		{
			Source:      `0\.0\.0\.0:\d+|INADDR_ANY`,
			PatternID:   "bind_all_interfaces",
			Severity:    "high",
			Category:    "network",
			Description: "binds to all network interfaces",
		},
		{
			Source:      `/bin/(ba)?sh\s+-i\s+.*>/dev/tcp/`,
			PatternID:   "bash_reverse_shell",
			Severity:    "critical",
			Category:    "network",
			Description: "bash interactive reverse shell via /dev/tcp",
		},
		{
			Source:      `python[23]?\s+-c\s+["']import\s+socket`,
			PatternID:   "python_socket_oneliner",
			Severity:    "critical",
			Category:    "network",
			Description: "Python one-liner socket connection (likely reverse shell)",
		},
		{
			Source:      `socket\.connect\s*\(\s*\(`,
			PatternID:   "python_socket_connect",
			Severity:    "high",
			Category:    "network",
			Description: "Python socket connect to arbitrary host",
		},
		{
			Source:      `webhook\.site|requestbin\.com|pipedream\.net|hookbin\.com`,
			PatternID:   "exfil_service",
			Severity:    "high",
			Category:    "network",
			Description: "references known data exfiltration/webhook testing service",
		},
		{
			Source:      `pastebin\.com|hastebin\.com|ghostbin\.`,
			PatternID:   "paste_service",
			Severity:    "medium",
			Category:    "network",
			Description: "references paste service (possible data staging)",
		},

		// -- Obfuscation: encoding and eval --
		{
			Source:      `base64\s+(-d|--decode)\s*\|`,
			PatternID:   "base64_decode_pipe",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "base64 decodes and pipes to execution",
		},
		{
			Source:      `\\x[0-9a-fA-F]{2}.*\\x[0-9a-fA-F]{2}.*\\x[0-9a-fA-F]{2}`,
			PatternID:   "hex_encoded_string",
			Severity:    "medium",
			Category:    "obfuscation",
			Description: "hex-encoded string (possible obfuscation)",
		},
		{
			Source:      `\beval\s*\(\s*["']`,
			PatternID:   "eval_string",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "eval() with string argument",
		},
		{
			Source:      `\bexec\s*\(\s*["']`,
			PatternID:   "exec_string",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "exec() with string argument",
		},
		{
			Source:      `echo\s+[^\n]*\|\s*(bash|sh|python|perl|ruby|node)`,
			PatternID:   "echo_pipe_exec",
			Severity:    "critical",
			Category:    "obfuscation",
			Description: "echo piped to interpreter for execution",
		},
		{
			Source:      `compile\s*\(\s*[^\)]+,\s*["'].*["']\s*,\s*["']exec["']\s*\)`,
			PatternID:   "python_compile_exec",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "Python compile() with exec mode",
		},
		{
			Source:      `getattr\s*\(\s*__builtins__`,
			PatternID:   "python_getattr_builtins",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "dynamic access to Python builtins (evasion technique)",
		},
		{
			Source:      `__import__\s*\(\s*["']os["']\s*\)`,
			PatternID:   "python_import_os",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "dynamic import of os module",
		},
		{
			Source:      `codecs\.decode\s*\(\s*["']`,
			PatternID:   "python_codecs_decode",
			Severity:    "medium",
			Category:    "obfuscation",
			Description: "codecs.decode (possible ROT13 or encoding obfuscation)",
		},
		{
			Source:      `String\.fromCharCode|charCodeAt`,
			PatternID:   "js_char_code",
			Severity:    "medium",
			Category:    "obfuscation",
			Description: "JavaScript character code construction (possible obfuscation)",
		},
		{
			Source:      `atob\s*\(|btoa\s*\(`,
			PatternID:   "js_base64",
			Severity:    "medium",
			Category:    "obfuscation",
			Description: "JavaScript base64 encode/decode",
		},
		{
			Source:      `\[::-1\]`,
			PatternID:   "string_reversal",
			Severity:    "low",
			Category:    "obfuscation",
			Description: "string reversal (possible obfuscated payload)",
		},
		{
			Source:      `chr\s*\(\s*\d+\s*\)\s*\+\s*chr\s*\(\s*\d+`,
			PatternID:   "chr_building",
			Severity:    "high",
			Category:    "obfuscation",
			Description: "building string from chr() calls (obfuscation)",
		},
		{
			Source:      `\\u[0-9a-fA-F]{4}.*\\u[0-9a-fA-F]{4}.*\\u[0-9a-fA-F]{4}`,
			PatternID:   "unicode_escape_chain",
			Severity:    "medium",
			Category:    "obfuscation",
			Description: "chain of unicode escapes (possible obfuscation)",
		},

		// -- Process execution in scripts --
		{
			Source:      `subprocess\.(run|call|Popen|check_output)\s*\(`,
			PatternID:   "python_subprocess",
			Severity:    "medium",
			Category:    "execution",
			Description: "Python subprocess execution",
		},
		{
			Source:      `os\.system\s*\(`,
			PatternID:   "python_os_system",
			Severity:    "high",
			Category:    "execution",
			Description: "os.system() — unguarded shell execution",
		},
		{
			Source:      `os\.popen\s*\(`,
			PatternID:   "python_os_popen",
			Severity:    "high",
			Category:    "execution",
			Description: "os.popen() — shell pipe execution",
		},
		{
			Source:      `child_process\.(exec|spawn|fork)\s*\(`,
			PatternID:   "node_child_process",
			Severity:    "high",
			Category:    "execution",
			Description: "Node.js child_process execution",
		},
		{
			Source:      `Runtime\.getRuntime\(\)\.exec\(`,
			PatternID:   "java_runtime_exec",
			Severity:    "high",
			Category:    "execution",
			Description: "Java Runtime.exec() — shell execution",
		},
		{
			Source:      "`[^`]*\\$\\([^)]+\\)[^`]*`",
			PatternID:   "backtick_subshell",
			Severity:    "medium",
			Category:    "execution",
			Description: "backtick string with command substitution",
		},

		// -- Path traversal --
		{
			Source:      `\.\./\.\./\.\.`,
			PatternID:   "path_traversal_deep",
			Severity:    "high",
			Category:    "traversal",
			Description: "deep relative path traversal (3+ levels up)",
		},
		{
			Source:      `\.\./\.\.`,
			PatternID:   "path_traversal",
			Severity:    "medium",
			Category:    "traversal",
			Description: "relative path traversal (2+ levels up)",
		},
		{
			Source:      `/etc/passwd|/etc/shadow`,
			PatternID:   "system_passwd_access",
			Severity:    "critical",
			Category:    "traversal",
			Description: "references system password files",
		},
		{
			Source:      `/proc/self|/proc/\d+/`,
			PatternID:   "proc_access",
			Severity:    "high",
			Category:    "traversal",
			Description: "references /proc filesystem (process introspection)",
		},
		{
			Source:      `/dev/shm/`,
			PatternID:   "dev_shm",
			Severity:    "medium",
			Category:    "traversal",
			Description: "references shared memory (common staging area)",
		},

		// -- Crypto mining --
		{
			Source:      `xmrig|stratum\+tcp|monero|coinhive|cryptonight`,
			PatternID:   "crypto_mining",
			Severity:    "critical",
			Category:    "mining",
			Description: "cryptocurrency mining reference",
		},
		{
			Source:      `hashrate|nonce.*difficulty`,
			PatternID:   "mining_indicators",
			Severity:    "medium",
			Category:    "mining",
			Description: "possible cryptocurrency mining indicators",
		},

		// -- Supply chain: curl/wget pipe to shell --
		{
			Source:      `curl\s+[^\n]*\|\s*(ba)?sh`,
			PatternID:   "curl_pipe_shell",
			Severity:    "critical",
			Category:    "supply_chain",
			Description: "curl piped to shell (download-and-execute)",
		},
		{
			Source:      `wget\s+[^\n]*-O\s*-\s*\|\s*(ba)?sh`,
			PatternID:   "wget_pipe_shell",
			Severity:    "critical",
			Category:    "supply_chain",
			Description: "wget piped to shell (download-and-execute)",
		},
		{
			Source:      `curl\s+[^\n]*\|\s*python`,
			PatternID:   "curl_pipe_python",
			Severity:    "critical",
			Category:    "supply_chain",
			Description: "curl piped to Python interpreter",
		},

		// -- Supply chain: unpinned/deferred dependencies --
		{
			Source:      `#\s*///\s*script.*dependencies`,
			PatternID:   "pep723_inline_deps",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "PEP 723 inline script metadata with dependencies (verify pinning)",
		},
		{
			Source:      `pip\s+install\s+`,
			PatternID:   "unpinned_pip_install",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "pip install without version pinning",
		},
		{
			Source:      `npm\s+install\s+`,
			PatternID:   "unpinned_npm_install",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "npm install without version pinning",
		},
		{
			Source:      `uv\s+run\s+`,
			PatternID:   "uv_run",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "uv run (may auto-install unpinned dependencies)",
		},

		// -- Supply chain: remote resource fetching --
		{
			Source:      `(curl|wget|httpx?\.get|requests\.get|fetch)\s*[\(]?\s*["']https?://`,
			PatternID:   "remote_fetch",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "fetches remote resource at runtime",
		},
		{
			Source:      `git\s+clone\s+`,
			PatternID:   "git_clone",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "clones a git repository at runtime",
		},
		{
			Source:      `docker\s+pull\s+`,
			PatternID:   "docker_pull",
			Severity:    "medium",
			Category:    "supply_chain",
			Description: "pulls a Docker image at runtime",
		},

		// -- Privilege escalation --
		{
			Source:      `^allowed-tools\s*:`,
			PatternID:   "allowed_tools_field",
			Severity:    "high",
			Category:    "privilege_escalation",
			Description: "skill declares allowed-tools (pre-approves tool access)",
		},
		{
			Source:      `\bsudo\b`,
			PatternID:   "sudo_usage",
			Severity:    "high",
			Category:    "privilege_escalation",
			Description: "uses sudo (privilege escalation)",
		},
		{
			Source:      `setuid|setgid|cap_setuid`,
			PatternID:   "setuid_setgid",
			Severity:    "critical",
			Category:    "privilege_escalation",
			Description: "setuid/setgid (privilege escalation mechanism)",
		},
		{
			Source:      `NOPASSWD`,
			PatternID:   "nopasswd_sudo",
			Severity:    "critical",
			Category:    "privilege_escalation",
			Description: "NOPASSWD sudoers entry (passwordless privilege escalation)",
		},
		{
			Source:      `chmod\s+[u+]?s`,
			PatternID:   "suid_bit",
			Severity:    "critical",
			Category:    "privilege_escalation",
			Description: "sets SUID/SGID bit on a file",
		},

		// -- Agent config persistence --
		{
			Source:      `AGENTS\.md|CLAUDE\.md|\.cursorrules|\.clinerules`,
			PatternID:   "agent_config_mod",
			Severity:    "critical",
			Category:    "persistence",
			Description: "references agent config files (could persist malicious instructions across sessions)",
		},
		{
			Source:      `\.hermes/config\.yaml|\.hermes/SOUL\.md`,
			PatternID:   "hermes_config_mod",
			Severity:    "critical",
			Category:    "persistence",
			Description: "references Hermes configuration files directly",
		},
		{
			Source:      `\.claude/settings|\.codex/config`,
			PatternID:   "other_agent_config",
			Severity:    "high",
			Category:    "persistence",
			Description: "references other agent configuration files",
		},

		// -- Hardcoded secrets --
		{
			Source:      `(?:api[_-]?key|token|secret|password)\s*[=:]\s*["'][A-Za-z0-9+/=_-]{20,}`,
			PatternID:   "hardcoded_secret",
			Severity:    "critical",
			Category:    "credential_exposure",
			Description: "possible hardcoded API key, token, or secret",
		},
		{
			Source:      `-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`,
			PatternID:   "embedded_private_key",
			Severity:    "critical",
			Category:    "credential_exposure",
			Description: "embedded private key",
		},
		{
			Source:      `ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{80,}`,
			PatternID:   "github_token_leaked",
			Severity:    "critical",
			Category:    "credential_exposure",
			Description: "GitHub personal access token in skill content",
		},
		{
			Source:      `sk-[A-Za-z0-9]{20,}`,
			PatternID:   "openai_key_leaked",
			Severity:    "critical",
			Category:    "credential_exposure",
			Description: "possible OpenAI API key in skill content",
		},
		{
			Source:      `sk-ant-[A-Za-z0-9_-]{90,}`,
			PatternID:   "anthropic_key_leaked",
			Severity:    "critical",
			Category:    "credential_exposure",
			Description: "possible Anthropic API key in skill content",
		},
		{
			Source:      `AKIA[0-9A-Z]{16}`,
			PatternID:   "aws_access_key_leaked",
			Severity:    "critical",
			Category:    "credential_exposure",
			Description: "AWS access key ID in skill content",
		},

		// -- Additional prompt injection: jailbreak patterns --
		{
			Source:      `\bDAN\s+mode\b|Do\s+Anything\s+Now`,
			PatternID:   "jailbreak_dan",
			Severity:    "critical",
			Category:    "injection",
			Description: "DAN (Do Anything Now) jailbreak attempt",
		},
		{
			Source:      `\bdeveloper\s+mode\b.*\benabled?\b`,
			PatternID:   "jailbreak_dev_mode",
			Severity:    "critical",
			Category:    "injection",
			Description: "developer mode jailbreak attempt",
		},
		{
			Source:      `hypothetical\s+scenario.*(?:ignore|bypass|override)`,
			PatternID:   "hypothetical_bypass",
			Severity:    "high",
			Category:    "injection",
			Description: "hypothetical scenario used to bypass restrictions",
		},
		{
			Source:      `for\s+educational\s+purposes?\s+only`,
			PatternID:   "educational_pretext",
			Severity:    "medium",
			Category:    "injection",
			Description: "educational pretext often used to justify harmful content",
		},
		{
			Source:      `(respond|answer|reply)\s+without\s+(?:\w+\s+)*(restrictions|limitations|filters|safety)`,
			PatternID:   "remove_filters",
			Severity:    "critical",
			Category:    "injection",
			Description: "instructs agent to respond without safety filters",
		},
		{
			Source:      `you\s+have\s+been\s+(?:\w+\s+)*(updated|upgraded|patched)\s+to`,
			PatternID:   "fake_update",
			Severity:    "high",
			Category:    "injection",
			Description: "fake update/patch announcement (social engineering)",
		},
		{
			Source:      `new\s+(?:\w+\s+)*policy|updated\s+(?:\w+\s+)*guidelines|revised\s+(?:\w+\s+)*instructions`,
			PatternID:   "fake_policy",
			Severity:    "medium",
			Category:    "injection",
			Description: "claims new policy/guidelines (may be social engineering)",
		},

		// -- Context window exfiltration --
		{
			Source:      `(include|output|print|send|share)\s+(?:\w+\s+)*(conversation|chat\s+history|previous\s+messages|context)`,
			PatternID:   "context_exfil",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "instructs agent to output/share conversation history",
		},
		{
			Source:      `(send|post|upload|transmit)\s+.*\s+(to|at)\s+https?://`,
			PatternID:   "send_to_url",
			Severity:    "high",
			Category:    "exfiltration",
			Description: "instructs agent to send data to a URL",
		},
	}

	for _, p := range rawPatterns {
		r, err := regexp.Compile("(?i)" + p.Source)
		if err != nil {
			log.Printf("invalid regex %q: %v", p.Source, err)
			continue
		}
		SkillGuardThreatPatterns = append(SkillGuardThreatPatterns, ThreatPattern{
			Regex:       r,
			PatternID:   p.PatternID,
			Severity:    p.Severity,
			Category:    p.Category,
			Description: p.Description,
		})
	}
}
