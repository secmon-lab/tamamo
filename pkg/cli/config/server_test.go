package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/cli/config"
)

func TestValidateTLS(t *testing.T) {
	t.Run("no TLS flags", func(t *testing.T) {
		cfg := config.Server{Addr: "127.0.0.1:8080"}
		gt.NoError(t, cfg.ValidateTLS())
	})

	t.Run("tls enabled without cert files", func(t *testing.T) {
		cfg := config.Server{Addr: "127.0.0.1:8080", TLS: true}
		gt.NoError(t, cfg.ValidateTLS())
	})

	t.Run("tls enabled with cert and key", func(t *testing.T) {
		cfg := config.Server{
			Addr:    "127.0.0.1:8080",
			TLS:     true,
			TLSCert: "/path/to/cert.pem",
			TLSKey:  "/path/to/key.pem",
		}
		gt.NoError(t, cfg.ValidateTLS())
	})

	t.Run("tls-cert only without tls-key is error", func(t *testing.T) {
		cfg := config.Server{
			Addr:    "127.0.0.1:8080",
			TLS:     true,
			TLSCert: "/path/to/cert.pem",
		}
		err := cfg.ValidateTLS()
		gt.Value(t, err != nil).Equal(true)
	})

	t.Run("tls-key only without tls-cert is error", func(t *testing.T) {
		cfg := config.Server{
			Addr:   "127.0.0.1:8080",
			TLS:    true,
			TLSKey: "/path/to/key.pem",
		}
		err := cfg.ValidateTLS()
		gt.Value(t, err != nil).Equal(true)
	})

	t.Run("tls-cert and tls-key without tls flag is error", func(t *testing.T) {
		cfg := config.Server{
			Addr:    "127.0.0.1:8080",
			TLSCert: "/path/to/cert.pem",
			TLSKey:  "/path/to/key.pem",
		}
		err := cfg.ValidateTLS()
		gt.Value(t, err != nil).Equal(true)
	})

	t.Run("tls-cert only without tls flag is error", func(t *testing.T) {
		cfg := config.Server{
			Addr:    "127.0.0.1:8080",
			TLSCert: "/path/to/cert.pem",
		}
		err := cfg.ValidateTLS()
		gt.Value(t, err != nil).Equal(true)
	})

	t.Run("tls-key only without tls flag is error", func(t *testing.T) {
		cfg := config.Server{
			Addr:   "127.0.0.1:8080",
			TLSKey: "/path/to/key.pem",
		}
		err := cfg.ValidateTLS()
		gt.Value(t, err != nil).Equal(true)
	})
}
