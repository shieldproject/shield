package agent_test

// SSH-channel probe for golang.org/x/crypto bumps.
//
// SHIELD imports golang.org/x/crypto/ssh first-party in BOTH roles: the
// shield-agent is an SSH server (agent/agent.go, agent/config.go) and the core
// is an SSH client (core/core.go, core/fabric/legacy.go). Nothing else in the
// repo exercises that channel end-to-end with the production client config:
// `make api-tests` starts a shield-agent but never runs a task against it, and
// the ginkgo "SSH Server" spec dials with a test-local client config that pins
// no MACs and asserts only that output is non-empty.
//
// This file closes that gap. It:
//   1. stands up the REAL shield-agent (agent.NewAgent + ReadConfig + ServeOne)
//      on an ephemeral loopback port, using the checked-in agent/test fixtures;
//   2. drives a real task through core's OWN client code path
//      (core/fabric.Legacy(...).Status(...)) using a ClientConfig that mirrors
//      core/core.go:314-319 exactly;
//   3. fingerprints the negotiated KEX / host-key / cipher / MAC so an
//      unintended downgrade is visible;
//   4. asserts auth still fails closed for an unauthorized key and for an SSH
//      certificate, and that the agent survives both.
//
// Run:  go test -mod=vendor ./agent/ -run TestXCryptoProbe -v
//
// No shieldd, no Vault, no BOSH, no docker. The "status" op needs no plugins
// beyond the checked-in agent/test/bin and needs neither shield-crypt nor
// shield-report, so this does not inherit the pre-existing PATH/exit-127
// failures of the ginkgo suite.

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shieldproject/shield/agent"
	"github.com/shieldproject/shield/core"
	"github.com/shieldproject/shield/core/fabric"
	"github.com/shieldproject/shield/db"
	"golang.org/x/crypto/ssh"
)

/* ------------------------------------------------------------------ *
 * Fixtures and production-faithful helpers
 * ------------------------------------------------------------------ */

const (
	authorizedKeyPath   = "test/identities/a/id_rsa"      // IS in test/authorized_keys
	unauthorizedKeyPath = "test/identities/server/id_rsa" // is NOT in test/authorized_keys
	probeAgentConf      = "test/test.conf"
)

// probeFingerprintBaseline is the negotiated SSH crypto between core and a
// shield-agent, plus the fixture host key's SHA256.
//
// Measured 2026-07-20 on BOTH sides of the Q3 2026 x/crypto bump and found
// byte-identical: tag v9.0.1 (x/crypto v0.50.0) and the bumped tree
// (v0.52.0) negotiate exactly this. The security release changed no
// algorithm on the wire, which is what makes pinning this safe.
//
// mac= is empty by design: aes128-gcm@openssh.com is AEAD, so integrity
// comes from the cipher and no separate MAC is negotiated. That is also why
// SHIELD's hmac-sha1 pin is dormant rather than live -- see the AEAD
// assertions above.
//
// If this must change, change it in the same commit as the thing that moved
// it, and say why in the cycle doc.
const probeFingerprintBaseline = "kex=mlkem768x25519-sha256 " +
	"hostkey=rsa-sha2-256 cipher=aes128-gcm@openssh.com mac= " +
	"hostkeyfp=SHA256:uS3hiuc/vNbEIoqiw7c/Phb8EVgagCtWi3PkG1JwVVo"

// coreMACs is the MAC list core negotiates with legacy agents. It reads core's
// own exported default (core/core.go), not a copy, so the inventory and
// negative-control probes cannot silently pass against a list core no longer
// uses. If an operator overrides macs in config, core uses that instead; this
// probe asserts the shipped default.
var coreMACs = core.DefaultLegacyAgentMACs

// coreClientConfig mirrors core/core.go:314-319 verbatim.
func coreClientConfig(t *testing.T, keyPath string) *ssh.ClientConfig {
	t.Helper()

	raw, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading %s: %s", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		t.Fatalf("ssh.ParsePrivateKey(%s): %s", keyPath, err)
	}

	return &ssh.ClientConfig{
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
		Config:          ssh.Config{MACs: coreMACs},
	}
}

// probePath makes the probe independent of run order. agent/command.go:150
// execs `shield-pipe` by bare name off PATH; the ginkgo suite amends PATH from
// inside TestAgent (agent_suite_test.go:23), which never runs under a
// `-run TestXCryptoProbe` filter. Without this, the handshake succeeds, the
// channel works, and the task still fails with the agent's magic rc 16777216 --
// a PATH fault wearing an SSH fault's clothes. Mirrors Makefile:25.
var probePathOnce sync.Once

func probePath(t *testing.T) {
	t.Helper()
	probePathOnce.Do(func() {
		wd, err := os.Getwd() // = <repo>/agent under `go test`
		if err != nil {
			t.Fatalf("getwd: %s", err)
		}
		os.Setenv("PATH", strings.Join([]string{
			wd + "/..", wd + "/../bin", wd + "/test/bin", os.Getenv("PATH"),
		}, ":"))
	})
}

// startProbeAgent boots the real shield-agent from agent/test/test.conf on an
// ephemeral port and serves connections until the test tears it down.
//
// It serves an UNBOUNDED number of connections rather than a caller-supplied
// count: encoding "this probe makes exactly N dials" bakes in the assumption
// that each logical dial opens exactly one TCP connection, so a future
// x/crypto that retries a handshake would starve the agent and the NEXT
// probe's dial would hang -- an SSH regression surfacing as an unrelated
// timeout. The loop exits when Cleanup closes the listener (Accept errors,
// ServeOne returns); the `closed` signal keeps that from hot-spinning on the
// error return (agent/agent.go:70-74).
//
// test.conf hard-codes 127.0.0.1:9122 and agent/config.go binds the listener
// inside ReadConfig, so a second bind in this test binary would EADDRINUSE
// against the ginkgo suite. SHIELD_AGENT_LISTEN_ADDRESS (agent/config.go:26) is
// applied by env.Override AFTER the YAML unmarshal, so it wins; it is unset
// again immediately so no other spec in this binary sees it.
func startProbeAgent(t *testing.T) string {
	t.Helper()
	probePath(t)

	os.Setenv("SHIELD_AGENT_LISTEN_ADDRESS", "127.0.0.1:0")
	ag := agent.NewAgent()
	err := ag.ReadConfig(probeAgentConf)
	os.Unsetenv("SHIELD_AGENT_LISTEN_ADDRESS")
	if err != nil {
		t.Fatalf("agent.ReadConfig(%s): %s", probeAgentConf, err)
	}

	addr := ag.Listen.Addr().String()
	closed := make(chan struct{})
	t.Cleanup(func() { close(closed); ag.Listen.Close() })
	go func() {
		for {
			ag.ServeOne(ag.Listen, false)
			select {
			case <-closed:
				return
			default:
			}
		}
	}()

	return addr
}

func negotiated(t *testing.T, c *ssh.Client) ssh.NegotiatedAlgorithms {
	t.Helper()
	m, ok := c.Conn.(ssh.AlgorithmsConnMetadata)
	if !ok {
		t.Fatal("FAIL: x/crypto no longer exposes ssh.AlgorithmsConnMetadata; " +
			"re-point the fingerprint, do not delete it")
	}
	return m.Algorithms()
}

/* ------------------------------------------------------------------ *
 * PROBE 1 -- production round trip through core's own client code
 * ------------------------------------------------------------------ */

func TestXCryptoProbeCoreToAgentRoundTrip(t *testing.T) {
	addr := startProbeAgent(t)
	cc := coreClientConfig(t, authorizedKeyPath)

	chore := fabric.Legacy(addr, cc, nil).Status(&db.Task{UUID: "xcrypto-probe"})

	// Single-goroutine drain, no mutex, no settle-sleep. Chore.Stdout/Stderr
	// are UNBUFFERED (core/scheduler/chore.go:38-40) and are never closed on
	// the chore.Do() path -- only the worker's run() closes them
	// (chore.go:157-158), which this test does not invoke. So a range/WaitGroup
	// join would block forever; that is why the original reached for a sleep.
	//
	// Because the channels are unbuffered, every send in fabric's Execute
	// blocks until received here, and UnixExit (which fires Exit) is called
	// only after the last Stdout/Stderr send returns (core/fabric/legacy.go:196
	// -202, guarded by <-wait). Receiving Exit in the same select therefore
	// happens-after every output write is received and appended -- deterministic
	// with no clock.
	var stdout, stderr strings.Builder
	go chore.Do(chore)

	var rc int
	deadline := time.After(60 * time.Second)
drain:
	for {
		select {
		case s := <-chore.Stdout:
			stdout.WriteString(s)
		case s := <-chore.Stderr:
			stderr.WriteString(s)
		case rc = <-chore.Exit:
			break drain
		case <-deadline:
			t.Fatal("FAIL: chore did not complete in 60s (handshake hung or agent wedged)")
		}
	}

	out, errs := stdout.String(), stderr.String()

	t.Logf("STDOUT (O: stream, demultiplexed by core/fabric/legacy.go):\n%s", out)
	t.Logf("STDERR (E: stream + core's own progress lines):\n%s", errs)

	if rc != 0 {
		t.Fatalf("FAIL: core->agent status task exited %d, want 0", rc)
	}
	// Proves the exec payload was parsed, shield-pipe ran, and the O: stream was
	// demultiplexed intact -- not merely that a connection was opened.
	for _, want := range []string{`"health"`, `"ok"`, `"name"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("FAIL: agent health JSON missing %s in the O: stream", want)
		}
	}
	// Proves the E: stream demultiplexed too, and that core's client really dialed.
	for _, want := range []string{"connecting to " + addr, "connected to " + addr} {
		if !strings.Contains(errs, want) {
			t.Fatalf("FAIL: expected %q on the E: stream", want)
		}
	}
	t.Log("PASS: core->agent SSH channel intact " +
		"(handshake, userauth, session, exec, O:/E: demux, exit-status)")
}

/* ------------------------------------------------------------------ *
 * PROBE 2 -- crypto fingerprint
 * ------------------------------------------------------------------ */

// aeadCiphers negotiate no separate MAC; a populated MAC under one of these is
// itself a signal.
var aeadCiphers = map[string]bool{
	"aes128-gcm@openssh.com":        true,
	"aes256-gcm@openssh.com":        true,
	"chacha20-poly1305@openssh.com": true,
}

func TestXCryptoProbeFingerprint(t *testing.T) {
	addr := startProbeAgent(t)
	cc := coreClientConfig(t, authorizedKeyPath)

	// Core uses InsecureIgnoreHostKey, so host-key drift is invisible in
	// production. Record it here instead of ignoring it.
	var hostKeyType, hostKeyFP string
	cc.HostKeyCallback = func(host string, remote net.Addr, key ssh.PublicKey) error {
		hostKeyType = key.Type()
		hostKeyFP = ssh.FingerprintSHA256(key)
		return nil
	}

	conn, err := ssh.Dial("tcp4", addr, cc)
	if err != nil {
		t.Fatalf("FAIL: dial %s: %s", addr, err)
	}
	defer conn.Close()

	a := negotiated(t, conn)

	// Prove the measured connection is the one that actually carries work.
	sess, err := conn.NewSession()
	if err != nil {
		t.Fatalf("FAIL: NewSession: %s", err)
	}
	out, err := sess.Output(`{"operation":"status"}`)
	sess.Close()
	if err != nil {
		t.Fatalf("FAIL: exec on the measured connection: %s (output %q)", err, out)
	}

	fp := fmt.Sprintf("kex=%s hostkey=%s cipher=%s mac=%s hostkeyfp=%s",
		a.KeyExchange, a.HostKey, a.Read.Cipher, a.Read.MAC, hostKeyFP)
	t.Logf("FINGERPRINT %s", fp)
	t.Logf("HOSTKEY type=%s fp=%s", hostKeyType, hostKeyFP)
	t.Logf("SERVER-VERSION %s", string(conn.ServerVersion()))

	/* --- hard assertions: things that must never regress --- */

	if a.Read.Cipher != a.Write.Cipher || a.Read.MAC != a.Write.MAC {
		t.Fatalf("FAIL: asymmetric negotiation r=%s/%s w=%s/%s",
			a.Read.Cipher, a.Read.MAC, a.Write.Cipher, a.Write.MAC)
	}
	if a.KeyExchange == "" {
		t.Fatal("FAIL: empty negotiated key exchange")
	}
	// CVE-2026-46597 lives in the AES-GCM packet decoder. SHIELD negotiates
	// AES-GCM by default, so that is live code, not a dormant path.
	if !aeadCiphers[a.Read.Cipher] {
		t.Errorf("ESCALATE: negotiated cipher %q is not AEAD. SHIELD's MAC pin "+
			"(core/core.go:312, agent/config.go:214) is now LIVE and still "+
			"contains hmac-sha1. negotiated mac=%q", a.Read.Cipher, a.Read.MAC)
	}
	if aeadCiphers[a.Read.Cipher] && a.Read.MAC != "" {
		t.Errorf("ESCALATE: AEAD cipher %q negotiated a separate MAC %q",
			a.Read.Cipher, a.Read.MAC)
	}
	// The fixtures are RSA. Negotiating up to rsa-sha2-* rather than SHA-1
	// ssh-rsa is the property that keeps them safe.
	if a.HostKey == "ssh-rsa" || a.HostKey == "ssh-dss" {
		t.Errorf("FAIL: host key algorithm downgraded to SHA-1 %q", a.HostKey)
	}

	/* --- drift gate: ON BY DEFAULT --- *
	 *
	 * probeFingerprintBaseline was measured on BOTH sides of the Q3 2026
	 * x/crypto v0.50.0 -> v0.52.0 bump and is byte-identical across it: the
	 * security release changed no negotiated algorithm. That is what makes it
	 * safe to pin. An env override exists for an INTENDED, justified move --
	 * it must not be used to silence an unexplained one.
	 *
	 * This gate is deliberately not opt-in. An env-var-only check never fires
	 * under `make go-tests`, which would make the "must match" claim in the
	 * cycle doc false -- precisely the doc-vs-artifact defect class this
	 * upgrade cycle exists to close. */
	want := probeFingerprintBaseline
	if env := os.Getenv("SHIELD_PROBE_FINGERPRINT"); env != "" {
		want = env
		t.Logf("baseline overridden from SHIELD_PROBE_FINGERPRINT")
	}
	if want != fp {
		t.Fatalf("FINGERPRINT DRIFT\n  expected: %s\n    actual: %s\n"+
			"The negotiated SSH crypto moved. If the move is intended, record "+
			"both lines in the cycle doc, justify it, and update "+
			"probeFingerprintBaseline in the same change.",
			want, fp)
	}
}

/* ------------------------------------------------------------------ *
 * PROBE 3 -- the library still offers everything SHIELD names
 * ------------------------------------------------------------------ */

func TestXCryptoProbeAlgorithmInventory(t *testing.T) {
	sup := ssh.SupportedAlgorithms()
	ins := ssh.InsecureAlgorithms()

	t.Logf("SUPPORTED macs=%v", sup.MACs)
	t.Logf("SUPPORTED ciphers=%v", sup.Ciphers)
	t.Logf("SUPPORTED kex=%v", sup.KeyExchanges)
	t.Logf("INSECURE  macs=%v hostkeys=%v pubkeyauths=%v",
		ins.MACs, ins.HostKeys, ins.PublicKeyAuths)

	in := func(hay []string, needle string) bool {
		for _, s := range hay {
			if s == needle {
				return true
			}
		}
		return false
	}

	// Both core (core.go:312) and the agent (config.go:214) hard-code this list.
	// If x/crypto reclassifies any entry, SHIELD's config names an algorithm the
	// library will no longer negotiate -- a runtime break against an unpatched
	// peer that no round trip between two matched-version peers can reveal.
	for _, m := range coreMACs {
		if !in(sup.MACs, m) {
			t.Errorf("FAIL: SHIELD pins MAC %q but x/crypto no longer offers it by "+
				"default (insecure list: %v). core/core.go:312 and agent/config.go:214 "+
				"must be updated.", m, ins.MACs)
		}
	}
}

/* ------------------------------------------------------------------ *
 * PROBE 4 -- negative control: the MAC path is real and this harness
 *            can actually see a break in it
 * ------------------------------------------------------------------ */

// Under the default AEAD cipher the MAC pin is inert and never negotiated, so
// a happy-path round trip cannot tell a working MAC negotiation from a broken
// one. This forces a CTR cipher (no AEAD) to make the MAC load-bearing, and
// then offers a MAC the agent does not name. The dial MUST fail. If it ever
// succeeds, the harness -- not SHIELD -- is broken, and every other probe here
// is suspect.
func TestXCryptoProbeNegativeControl(t *testing.T) {
	addr := startProbeAgent(t)

	// (a) CTR + a MAC outside SHIELD's pinned triple => must NOT negotiate.
	bad := coreClientConfig(t, authorizedKeyPath)
	bad.Config = ssh.Config{
		Ciphers: []string{"aes128-ctr"},
		MACs:    []string{"hmac-sha2-512-etm@openssh.com"},
	}
	if conn, err := ssh.Dial("tcp4", addr, bad); err == nil {
		conn.Close()
		t.Fatal("HARNESS FAIL: a MAC the agent does not offer was negotiated anyway")
	} else {
		t.Logf("negative control fired as designed: %s", err)
	}

	// (b) CTR + SHIELD's pinned triple => must negotiate, and the MAC must now
	//     be populated. This is the only place SHIELD's MAC pin is observable.
	good := coreClientConfig(t, authorizedKeyPath)
	good.Config = ssh.Config{Ciphers: []string{"aes128-ctr"}, MACs: coreMACs}
	conn, err := ssh.Dial("tcp4", addr, good)
	if err != nil {
		t.Fatalf("FAIL: SHIELD's pinned MACs no longer negotiate under a CTR cipher: %s", err)
	}
	defer conn.Close()

	a := negotiated(t, conn)
	t.Logf("FINGERPRINT(mac-forced) kex=%s hostkey=%s cipher=%s mac=%s",
		a.KeyExchange, a.HostKey, a.Read.Cipher, a.Read.MAC)
	if a.Read.MAC == "" {
		t.Fatal("FAIL: expected a populated MAC under a non-AEAD cipher")
	}
}

/* ------------------------------------------------------------------ *
 * PROBE 5 -- auth fails closed, and the agent survives
 * ------------------------------------------------------------------ */

func TestXCryptoProbeUnauthorizedKeyRejected(t *testing.T) {
	addr := startProbeAgent(t)

	// server/id_rsa is the agent's own host key; it is NOT in test/authorized_keys.
	bad := coreClientConfig(t, unauthorizedKeyPath)
	if conn, err := ssh.Dial("tcp4", addr, bad); err == nil {
		conn.Close()
		t.Fatal("SECURITY FAIL: unauthorized key was accepted")
	} else {
		t.Logf("unauthorized key correctly rejected: %s", err)
	}

	// The agent must still be serving afterwards.
	conn, err := ssh.Dial("tcp4", addr, coreClientConfig(t, authorizedKeyPath))
	if err != nil {
		t.Fatalf("FAIL: agent did not survive a rejected auth: %s", err)
	}
	conn.Close()
	t.Log("PASS: auth fails closed and the agent stays up")
}

func TestXCryptoProbeCertificateRejectedCleanly(t *testing.T) {
	addr := startProbeAgent(t)

	raw, err := os.ReadFile(authorizedKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		t.Fatal(err)
	}

	// A throwaway CA signs a certificate over the OTHERWISE-AUTHORIZED key.
	// SHIELD's ssh.CertChecker (agent/config.go:197) sets IsUserAuthority ->
	// false and leaves IsHostAuthority/HostKeyFallback nil -- the shape
	// CVE-2026-39835 (cert checker panics when auth callbacks are not set)
	// describes. The agent runs in this process, so a panic in its handshake
	// takes this test down with it.
	caRSA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := ssh.NewSignerFromKey(caRSA)
	if err != nil {
		t.Fatal(err)
	}
	cert := &ssh.Certificate{
		Key:             signer.PublicKey(),
		CertType:        ssh.UserCert,
		KeyId:           "xcrypto-probe",
		ValidPrincipals: []string{"shield"},
		ValidAfter:      0,
		ValidBefore:     ssh.CertTimeInfinity,
	}
	if err := cert.SignCert(rand.Reader, ca); err != nil {
		t.Fatal(err)
	}
	certSigner, err := ssh.NewCertSigner(cert, signer)
	if err != nil {
		t.Fatal(err)
	}

	cc := &ssh.ClientConfig{
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(certSigner)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
		Config:          ssh.Config{MACs: coreMACs},
	}
	if conn, err := ssh.Dial("tcp4", addr, cc); err == nil {
		conn.Close()
		t.Fatal("SECURITY FAIL: a CA-signed user certificate was ACCEPTED; " +
			"agent/config.go IsUserAuthority=false must reject it")
	} else {
		t.Logf("certificate correctly rejected: %s", err)
	}

	conn, err := ssh.Dial("tcp4", addr, coreClientConfig(t, authorizedKeyPath))
	if err != nil {
		t.Fatalf("FAIL: agent did not survive a certificate offer "+
			"(panic in CertChecker?): %s", err)
	}
	conn.Close()
	t.Log("PASS: certificate rejected without panicking the agent")
}
