## Dependency Update Report

Generated: Sun Sep  6 12:25:59 UTC 2026

## Dependency Update Summary

Go version: 1.26.4

```diff
diff --git a/go.mod b/go.mod
index 071c1a3..ef7050c 100644
--- a/go.mod
+++ b/go.mod
@@ -9,7 +9,7 @@ require (
 )
 
 require (
-	github.com/andybalholm/cascadia v1.3.4 // indirect
+	github.com/andybalholm/cascadia v1.3.5 // indirect
 	github.com/inconshreveable/mousetrap v1.1.0 // indirect
 	github.com/spf13/pflag v1.0.10 // indirect
 	golang.org/x/net v0.55.0 // indirect
diff --git a/go.sum b/go.sum
index 72c3efd..f86175e 100644
--- a/go.sum
+++ b/go.sum
@@ -1,7 +1,7 @@
 github.com/PuerkitoBio/goquery v1.11.0 h1:jZ7pwMQXIITcUXNH83LLk+txlaEy6NVOfTuP43xxfqw=
 github.com/PuerkitoBio/goquery v1.11.0/go.mod h1:wQHgxUOU3JGuj3oD/QFfxUdlzW6xPHfqyHre6VMY4DQ=
-github.com/andybalholm/cascadia v1.3.4 h1:vM2lgh0Vru9Vwyfm4cQqWP2HHMW0u0+2PAW7Q38Qufg=
-github.com/andybalholm/cascadia v1.3.4/go.mod h1:BLRmbRjpEtNKieZOCCvYj4RqN+KRA41GBe/5O+G93kM=
+github.com/andybalholm/cascadia v1.3.5 h1:RLjq12WJy58dN6eCIQrz0bAGZkztHWsEPFxP53Y7Ms8=
+github.com/andybalholm/cascadia v1.3.5/go.mod h1:BLRmbRjpEtNKieZOCCvYj4RqN+KRA41GBe/5O+G93kM=
 github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
 github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
 github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
```

## Security Scan

⚠️ **Vulnerabilities found:**
```
=== Symbol Results ===

Vulnerability #1: GO-2026-6218
    Avoid quadratic complexity in resolvePath in net/url
  More info: https://pkg.go.dev/vuln/GO-2026-6218
  Standard library
    Found in: net/url@go1.26.4
    Fixed in: net/url@go1.26.6
    Example traces found:
      #1: internal/preferences/gist.go:317:24: preferences.CreateGist calls http.Client.Do, which eventually calls url.URL.Parse

Vulnerability #2: GO-2026-6090
    Limit handshake messages we are willing to accept post-handshake in
    crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2026-6090
  Standard library
    Found in: crypto/tls@go1.26.4
    Fixed in: crypto/tls@go1.26.6
    Example traces found:
      #1: internal/preferences/gist.go:317:24: preferences.CreateGist calls http.Client.Do, which eventually calls tls.Conn.HandshakeContext
      #2: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls tls.Conn.Read
      #3: cmd/vga-events-telegram/main.go:386:14: vga.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/preferences/gist.go:317:24: preferences.CreateGist calls http.Client.Do, which eventually calls tls.Dialer.DialContext

Vulnerability #3: GO-2026-5972
    Enforce maximum recursion depth in encoding/asn1
  More info: https://pkg.go.dev/vuln/GO-2026-5972
  Standard library
    Found in: encoding/asn1@go1.26.4
    Fixed in: encoding/asn1@go1.26.6
    Example traces found:
      #1: internal/preferences/gist.go:321:2: preferences.CreateGist calls http.cancelTimerBody.Close, which eventually calls asn1.Unmarshal

Vulnerability #4: GO-2026-5856
    Invoking Encrypted Client Hello privacy leak in crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2026-5856
  Standard library
    Found in: crypto/tls@go1.26.4
    Fixed in: crypto/tls@go1.26.5
    Example traces found:
      #1: internal/preferences/gist.go:317:24: preferences.CreateGist calls http.Client.Do, which eventually calls tls.Conn.HandshakeContext
      #2: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls tls.Conn.Read
      #3: cmd/vga-events-telegram/main.go:386:14: vga.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/preferences/gist.go:317:24: preferences.CreateGist calls http.Client.Do, which eventually calls tls.Dialer.DialContext

Vulnerability #5: GO-2026-5026
    Invoking failure to reject ASCII-only Punycode-encoded labels in
    golang.org/x/net/idna
  More info: https://pkg.go.dev/vuln/GO-2026-5026
  Standard library
    Found in: net/http@go1.26.4
    Fixed in: net/http@go1.26.6
    Example traces found:
      #1: internal/preferences/gist.go:317:24: preferences.CreateGist calls http.Client.Do
      #2: cmd/vga-events-bot/main.go:3035:25: vga.getUpdatesWithTimeout calls http.Client.Get

Your code is affected by 5 vulnerabilities from the Go standard library.
This scan also found 3 vulnerabilities in packages you import and 19
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.
```

## Test Results

✅ All tests pass with updated dependencies
