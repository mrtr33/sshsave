package askpass

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// When run as a child with this variable set, the test binary behaves like
// sshsave in askpass mode
func TestMain(m *testing.M) {
	if os.Getenv("ASKPASS_TEST_CHILD") == "1" {
		os.Exit(Run(os.Args[1:]))
	}
	os.Exit(m.Run())
}

func runChild(t *testing.T, env []string, prompt string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], prompt)
	cmd.Env = append(env, "ASKPASS_TEST_CHILD=1")
	// New session without a controlling terminal, so /dev/tty cannot be opened
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	out, err := cmd.Output()
	return string(out), err
}

func TestEnvSetsAskpass(t *testing.T) {
	env, err := Env("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, "\n")
	for _, want := range []string{"SSH_ASKPASS_REQUIRE=force", envMode + "=1", envSecret + "=s3cret"} {
		if !strings.Contains(joined, want) {
			t.Errorf("env missing %q", want)
		}
	}
}

func TestRunAnswersPasswordPrompt(t *testing.T) {
	env, _ := Env("s3cret")
	out, err := runChild(t, env, "me@host's password: ")
	if err != nil {
		t.Fatal(err)
	}
	if out != "s3cret\n" {
		t.Errorf("got %q", out)
	}
}

func TestRunNeverAnswersConfirmationWithSecret(t *testing.T) {
	env, _ := Env("s3cret")
	// No terminal is available in tests, so the confirmation must fail
	// rather than fall back to the secret
	out, _ := runChild(t, env, "Are you sure you want to continue connecting (yes/no/[fingerprint])? ")
	if strings.Contains(out, "s3cret") {
		t.Error("secret was used to answer a host key confirmation")
	}
}

func TestRunWithoutSecretFails(t *testing.T) {
	_, err := runChild(t, []string{envMode + "=1"}, "password: ")
	if err == nil {
		t.Error("expected failure without secret")
	}
}
