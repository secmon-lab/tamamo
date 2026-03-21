package tlscert_test

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/service/tlscert"
)

func TestGenerate(t *testing.T) {
	cert, err := tlscert.Generate()
	gt.NoError(t, err)
	gt.NotEqual(t, cert, (*tls.Certificate)(nil))
	gt.B(t, len(cert.Certificate) > 0).True()
	gt.B(t, cert.PrivateKey != nil).True()
}

func TestGenerateCertificateAttributes(t *testing.T) {
	cert, err := tlscert.Generate()
	gt.NoError(t, err)

	// Parse server certificate (first in chain)
	serverCert, err := x509.ParseCertificate(cert.Certificate[0])
	gt.NoError(t, err)

	// Verify OpenSSL-default subject attributes
	gt.Value(t, serverCert.Subject.Country).Equal([]string{"AU"})
	gt.Value(t, serverCert.Subject.Province).Equal([]string{"Some-State"})
	gt.Value(t, serverCert.Subject.Organization).Equal([]string{"Internet Widgits Pty Ltd"})
	gt.Value(t, serverCert.Subject.CommonName).Equal("localhost")

	// Verify SANs
	gt.Value(t, serverCert.DNSNames).Equal([]string{"localhost"})
	foundIPv4 := false
	for _, ip := range serverCert.IPAddresses {
		if ip.Equal(net.IPv4(127, 0, 0, 1)) {
			foundIPv4 = true
		}
	}
	gt.Value(t, foundIPv4).Equal(true)
}

func TestGenerateCertificateChain(t *testing.T) {
	cert, err := tlscert.Generate()
	gt.NoError(t, err)

	// Should have 2 certificates: server + CA
	gt.Value(t, len(cert.Certificate)).Equal(2)

	// Parse CA certificate (second in chain)
	caCert, err := x509.ParseCertificate(cert.Certificate[1])
	gt.NoError(t, err)
	gt.Value(t, caCert.IsCA).Equal(true)
	gt.Value(t, caCert.Subject.Organization).Equal([]string{"Internet Widgits Pty Ltd"})

	// Verify server cert is signed by CA
	serverCert, err := x509.ParseCertificate(cert.Certificate[0])
	gt.NoError(t, err)

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)
	_, err = serverCert.Verify(x509.VerifyOptions{
		Roots: caPool,
	})
	gt.NoError(t, err)
}

func TestGenerateNotBeforeIsBackdated(t *testing.T) {
	cert, err := tlscert.Generate()
	gt.NoError(t, err)

	serverCert, err := x509.ParseCertificate(cert.Certificate[0])
	gt.NoError(t, err)

	now := time.Now()
	threeMonthsAgo := now.AddDate(0, 0, -90)
	oneYearAgo := now.AddDate(0, 0, -365)

	// NotBefore should be between 1 year ago and 3 months ago
	gt.Value(t, serverCert.NotBefore.Before(threeMonthsAgo)).Equal(true)
	gt.Value(t, serverCert.NotBefore.After(oneYearAgo.Add(-24*time.Hour))).Equal(true)

	// NotAfter should be ~2 years from NotBefore
	expectedNotAfter := serverCert.NotBefore.Add(2 * 365 * 24 * time.Hour)
	diff := serverCert.NotAfter.Sub(expectedNotAfter)
	if diff < 0 {
		diff = -diff
	}
	gt.Value(t, diff < time.Minute).Equal(true)
}

func TestGenerateTLSHandshake(t *testing.T) {
	cert, err := tlscert.Generate()
	gt.NoError(t, err)

	// Start a TLS server
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	gt.NoError(t, err)
	defer func() { _ = listener.Close() }()

	tlsListener := tls.NewListener(listener, tlsConfig)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	defer func() { _ = server.Close() }()

	go func() { _ = server.Serve(tlsListener) }()

	// Build a client that trusts the generated CA
	caCert, err := x509.ParseCertificate(cert.Certificate[1])
	gt.NoError(t, err)

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caPool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	resp, err := client.Get("https://" + listener.Addr().String())
	gt.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	gt.Value(t, resp.StatusCode).Equal(http.StatusOK)
}
