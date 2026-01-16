// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"testing"
)

func TestFingerprint(t *testing.T) {
	emptyTests := []struct {
		name  string
		input []byte
	}{
		{"nil input", nil},
		{"empty input", []byte{}},
	}

	for _, tt := range emptyTests {
		t.Run(tt.name, func(t *testing.T) {
			fp := Fingerprint(tt.input)
			if fp != "" {
				t.Fatalf("expected empty string, got %s", fp)
			}
		})
	}

	t.Run("valid data", func(t *testing.T) {
		data := []byte("valid data")
		// To verify: echo -n "valid data" | openssl dgst -sha256 |
		//   awk '{print toupper($2)}' | sed 's/../&:/g;s/:$//'
		expected := "D6:3E:23:E8:A7:CB:E0:80:F2:A7:99:84:FB:4B:2E:08:D2:29:24:E0:F2:7F:A7:B3:02:20:E4:E3:51:48:99:62"
		fp := Fingerprint(data)
		if fp != expected {
			t.Fatalf("expected %s, got %s", expected, fp)
		}
	})
}

func TestCertificateFingerprint(t *testing.T) {
	errorTests := []struct {
		name  string
		input []byte
	}{
		{"nil input", nil},
		{"empty input", []byte{}},
		{"invalid pem", []byte("not a certificate")},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CertificateFingerprint(tt.input)
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}

	t.Run("valid certificate", func(t *testing.T) {
		certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIBezCCASWgAwIBAgIUaDRT+aTsIPW8AeFz2M3NE8dKT30wDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHdGVzdC1jYTAeFw0yNjAxMTYxMTIxNDFaFw0yNzAxMTYx
MTIxNDFaMBIxEDAOBgNVBAMMB3Rlc3QtY2EwXDANBgkqhkiG9w0BAQEFAANLADBI
AkEA4k5kAs1U4VT8jgYjy0G76p+Q6Tc22T6G/jtp8bMons9n+4E12ja60RyH99ur
Qdn69Dq7bXuqBsAVJx/zeWSL4QIDAQABo1MwUTAdBgNVHQ4EFgQUHFuQ5P3Fad4I
nqCK4TP++m5JEOswHwYDVR0jBBgwFoAUHFuQ5P3Fad4InqCK4TP++m5JEOswDwYD
VR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAH84Ih9Kdr+sOEdD6fg89xqF
9OtEyeUj7+XESxYwNzk62wJmawBiBg6/O0iIV0o4Z11KcST2KKdTGb1XwEcJaVI=
-----END CERTIFICATE-----`)

		// To verify: openssl x509 -noout -fingerprint -sha256 -in cert.pem | cut -d= -f2
		expected := "BA:A5:C7:3B:3F:6E:06:27:19:F5:45:FC:6F:07:42:81:3B:F6:4D:61:95:CC:D5:D8:79:22:65:63:35:63:97:00"

		fp, err := CertificateFingerprint(certPem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fp != expected {
			t.Fatalf("fingerprint mismatch:\nexpected: %s\ngot:      %s", expected, fp)
		}
	})
}
