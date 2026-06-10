package cli

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// makeCert builds a self-signed leaf certificate valid for the supplied DNS
// name (and IP, so dialing 127.0.0.1 still reaches the listener) plus a
// matching tls.Config and an x509 pool that trusts it.
func makeCert(t *testing.T, dnsName string, notAfter time.Time) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: dnsName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{dnsName},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(leaf)

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, pool
}

// startTLSServer spins up a TLS listener that completes handshakes using cert
// and returns its address. It is closed when the test finishes.
func startTLSServer(t *testing.T, cert tls.Certificate) string {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			go func() {
				// Drive the handshake so the client side completes, then drop.
				if tc, ok := conn.(*tls.Conn); ok {
					_ = tc.Handshake()
				}
				_ = conn.Close()
			}()
		}
	}()

	return ln.Addr().String()
}

func TestProbeFrontingTLS_ValidCert(t *testing.T) {
	cert, pool := makeCert(t, "front.example.org", time.Now().Add(24*time.Hour))
	addr := startTLSServer(t, cert)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := probeFrontingTLS(ctx, &net.Dialer{}, addr, "front.example.org", pool)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestProbeFrontingTLS_WrongHost(t *testing.T) {
	// Cert is for front.example.org, but we verify against other.example.org.
	cert, pool := makeCert(t, "front.example.org", time.Now().Add(24*time.Hour))
	addr := startTLSServer(t, cert)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := probeFrontingTLS(ctx, &net.Dialer{}, addr, "other.example.org", pool)
	if err == nil {
		t.Fatal("expected SAN-mismatch failure, got success")
	}
	if !strings.Contains(err.Error(), "x509") {
		t.Fatalf("expected x509 verification error, got: %v", err)
	}
}

func TestProbeFrontingTLS_UntrustedCA(t *testing.T) {
	// Server cert is self-signed; we hand the client an empty pool that does
	// not trust it. Default verification must reject.
	cert, _ := makeCert(t, "front.example.org", time.Now().Add(24*time.Hour))
	addr := startTLSServer(t, cert)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := probeFrontingTLS(ctx, &net.Dialer{}, addr, "front.example.org", x509.NewCertPool())
	if err == nil {
		t.Fatal("expected untrusted-CA failure, got success")
	}
	if !strings.Contains(err.Error(), "x509") {
		t.Fatalf("expected x509 verification error, got: %v", err)
	}
}

func TestProbeFrontingTLS_ExpiredCert(t *testing.T) {
	// Same trust anchor, but the cert is already expired.
	cert, pool := makeCert(t, "front.example.org", time.Now().Add(-time.Hour))
	addr := startTLSServer(t, cert)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := probeFrontingTLS(ctx, &net.Dialer{}, addr, "front.example.org", pool)
	if err == nil {
		t.Fatal("expected expiry failure, got success")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expiry error, got: %v", err)
	}
}

func TestProbeFrontingTLS_OverrideDialDifferentFromSNI(t *testing.T) {
	// Domain-fronting override: dial one address (the listener bound to
	// 127.0.0.1), but verify the cert against the secret host name. The cert
	// is issued for the secret host, so verification must pass even though the
	// dial target is a bare IP:port.
	cert, pool := makeCert(t, "secret.example.org", time.Now().Add(24*time.Hour))
	addr := startTLSServer(t, cert) // e.g. 127.0.0.1:NNNNN

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// dial-target (addr, an IP:port) != SNI (secret.example.org)
	err := probeFrontingTLS(ctx, &net.Dialer{}, addr, "secret.example.org", pool)
	if err != nil {
		t.Fatalf("expected success when dialing override addr with secret-host SNI, got: %v", err)
	}
}
