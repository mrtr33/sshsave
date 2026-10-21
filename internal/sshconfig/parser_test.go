package sshconfig

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const sampleConfig = `# comment
Host work
    HostName 10.0.0.1
    User me
    ProxyJump bastion

Host a b
    Port=2222

Host *.example.com
    User wild

Match host foo
    User matched

Host = eq
    IdentityFile "~/my keys/id"

Host !neg *
    ForwardAgent yes
`

func TestParseSSHConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(sampleConfig), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	var valid []string
	for _, e := range entries {
		if e.IsValidEntry() {
			valid = append(valid, e.Host)
		}
	}

	want := []string{"work", "a", "b", "eq"}
	if len(valid) != len(want) {
		t.Fatalf("valid hosts = %v, want %v", valid, want)
	}
	for i := range want {
		if valid[i] != want[i] {
			t.Fatalf("valid hosts = %v, want %v", valid, want)
		}
	}
}

func TestToSSHCommand(t *testing.T) {
	h := HostEntry{Host: "work"}
	if got := h.ToSSHCommand(""); got != "ssh work" {
		t.Errorf("got %q", got)
	}
	if got := h.ToSSHCommand("/tmp/my config"); got != "ssh -F '/tmp/my config' work" {
		t.Errorf("got %q", got)
	}
}

func TestShellQuoteIsSafeInBash(t *testing.T) {
	for _, s := range []string{"plain", "with space", "it's", "$(touch /tmp/pwned)", "a;b", "`x`"} {
		out, err := exec.Command("bash", "-c", "printf %s "+shellQuote(s)).Output()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != s {
			t.Errorf("shellQuote(%q) round-trips to %q", s, out)
		}
	}
}
