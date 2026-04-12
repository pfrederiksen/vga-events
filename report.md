## Dependency Update Report

Generated: Sun Apr 12 09:35:57 UTC 2026

## Dependency Update Summary

Go version: 1.24.13

```diff
diff --git a/go.mod b/go.mod
index f05c937..20f4bce 100644
--- a/go.mod
+++ b/go.mod
@@ -11,6 +11,6 @@ require (
 require (
 	github.com/andybalholm/cascadia v1.3.3 // indirect
 	github.com/inconshreveable/mousetrap v1.1.0 // indirect
-	github.com/spf13/pflag v1.0.9 // indirect
+	github.com/spf13/pflag v1.0.10 // indirect
 	golang.org/x/net v0.48.0 // indirect
 )
diff --git a/go.sum b/go.sum
index 84a32f7..bf5c0b4 100644
--- a/go.sum
+++ b/go.sum
@@ -9,8 +9,9 @@ github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLf
 github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
 github.com/spf13/cobra v1.10.2 h1:DMTTonx5m65Ic0GOoRY2c16WCbHxOOw6xxezuLaBpcU=
 github.com/spf13/cobra v1.10.2/go.mod h1:7C1pvHqHw5A4vrJfjNwvOdzYu0Gml16OCs2GRiTUUS4=
-github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=
 github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
+github.com/spf13/pflag v1.0.10 h1:4EBh2KAYBwaONj6b2Ye1GiHfwjqyROoF4RwYO+vPwFk=
+github.com/spf13/pflag v1.0.10/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
 github.com/yuin/goldmark v1.4.13/go.mod h1:6yULJ656Px+3vBD8DxQVa3kxgyrAnzto9xy5taEt/CY=
 go.yaml.in/yaml/v3 v3.0.4/go.mod h1:DhzuOOF2ATzADvBadXxruRBLzYTpT36CKvDb3+aBEFg=
 golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2/go.mod h1:djNgcEr1/C05ACkg1iLfiJU5Ep61QUkGW8qpdssI0+w=
```

## Security Scan

⚠️ **Vulnerabilities found:**
```
=== Symbol Results ===

Vulnerability #1: GO-2026-4947
    Unexpected work during chain building in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2026-4947
  Standard library
    Found in: crypto/x509@go1.24.13
    Fixed in: crypto/x509@go1.25.9
    Example traces found:
      #1: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls x509.Certificate.Verify

Vulnerability #2: GO-2026-4946
    Inefficient policy validation in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2026-4946
  Standard library
    Found in: crypto/x509@go1.24.13
    Fixed in: crypto/x509@go1.25.9
    Example traces found:
      #1: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls x509.Certificate.Verify

Vulnerability #3: GO-2026-4870
    Unauthenticated TLS 1.3 KeyUpdate record can cause persistent connection
    retention and DoS in crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2026-4870
  Standard library
    Found in: crypto/tls@go1.24.13
    Fixed in: crypto/tls@go1.25.9
    Example traces found:
      #1: internal/preferences/gist.go:217:24: preferences.CreateGist calls http.Client.Do, which eventually calls tls.Conn.HandshakeContext
      #2: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls tls.Conn.Read
      #3: cmd/vga-events-telegram/main.go:386:14: vga.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/preferences/gist.go:217:24: preferences.CreateGist calls http.Client.Do, which eventually calls tls.Dialer.DialContext

Vulnerability #4: GO-2026-4602
    FileInfo can escape from a Root in os
  More info: https://pkg.go.dev/vuln/GO-2026-4602
  Standard library
    Found in: os@go1.24.13
    Fixed in: os@go1.25.8
    Example traces found:
      #1: cmd/vga-events-telegram/main.go:17:58: vga.init calls os.Getenv, which eventually calls os.ReadDir

Vulnerability #5: GO-2026-4601
    Incorrect parsing of IPv6 host literals in net/url
  More info: https://pkg.go.dev/vuln/GO-2026-4601
  Standard library
    Found in: net/url@go1.24.13
    Fixed in: net/url@go1.25.8
    Example traces found:
      #1: internal/preferences/gist.go:207:29: preferences.CreateGist calls http.NewRequest, which eventually calls url.Parse
      #2: internal/preferences/gist.go:217:24: preferences.CreateGist calls http.Client.Do, which eventually calls url.URL.Parse

Your code is affected by 5 vulnerabilities from the Go standard library.
This scan also found 1 vulnerability in packages you import and 3
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.
```

## Test Results

✅ All tests pass with updated dependencies
