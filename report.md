## Dependency Update Report

Generated: Sun Jun  7 10:59:28 UTC 2026

## Dependency Update Summary

Go version: 1.26.2

```diff
diff --git a/go.mod b/go.mod
index 6e5b900..0819451 100644
--- a/go.mod
+++ b/go.mod
@@ -5,12 +5,12 @@ go 1.26.2
 require (
 	github.com/PuerkitoBio/goquery v1.11.0
 	github.com/spf13/cobra v1.10.2
-	golang.org/x/crypto v0.47.0
+	golang.org/x/crypto v0.51.0
 )
 
 require (
-	github.com/andybalholm/cascadia v1.3.3 // indirect
+	github.com/andybalholm/cascadia v1.3.4 // indirect
 	github.com/inconshreveable/mousetrap v1.1.0 // indirect
 	github.com/spf13/pflag v1.0.10 // indirect
-	golang.org/x/net v0.48.0 // indirect
+	golang.org/x/net v0.55.0 // indirect
 )
diff --git a/go.sum b/go.sum
index bf5c0b4..72c3efd 100644
--- a/go.sum
+++ b/go.sum
@@ -1,9 +1,8 @@
 github.com/PuerkitoBio/goquery v1.11.0 h1:jZ7pwMQXIITcUXNH83LLk+txlaEy6NVOfTuP43xxfqw=
 github.com/PuerkitoBio/goquery v1.11.0/go.mod h1:wQHgxUOU3JGuj3oD/QFfxUdlzW6xPHfqyHre6VMY4DQ=
-github.com/andybalholm/cascadia v1.3.3 h1:AG2YHrzJIm4BZ19iwJ/DAua6Btl3IwJX+VI4kktS1LM=
-github.com/andybalholm/cascadia v1.3.3/go.mod h1:xNd9bqTn98Ln4DwST8/nG+H0yuB8Hmgu1YHNnWw0GeA=
+github.com/andybalholm/cascadia v1.3.4 h1:vM2lgh0Vru9Vwyfm4cQqWP2HHMW0u0+2PAW7Q38Qufg=
+github.com/andybalholm/cascadia v1.3.4/go.mod h1:BLRmbRjpEtNKieZOCCvYj4RqN+KRA41GBe/5O+G93kM=
 github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
-github.com/google/go-cmp v0.6.0/go.mod h1:17dUlkBOakJ0+DkrSSNjCkIjxS6bF9zb3elmeNGIjoY=
 github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
 github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
 github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
@@ -12,73 +11,9 @@ github.com/spf13/cobra v1.10.2/go.mod h1:7C1pvHqHw5A4vrJfjNwvOdzYu0Gml16OCs2GRiT
 github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
 github.com/spf13/pflag v1.0.10 h1:4EBh2KAYBwaONj6b2Ye1GiHfwjqyROoF4RwYO+vPwFk=
 github.com/spf13/pflag v1.0.10/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
-github.com/yuin/goldmark v1.4.13/go.mod h1:6yULJ656Px+3vBD8DxQVa3kxgyrAnzto9xy5taEt/CY=
 go.yaml.in/yaml/v3 v3.0.4/go.mod h1:DhzuOOF2ATzADvBadXxruRBLzYTpT36CKvDb3+aBEFg=
-golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2/go.mod h1:djNgcEr1/C05ACkg1iLfiJU5Ep61QUkGW8qpdssI0+w=
-golang.org/x/crypto v0.0.0-20210921155107-089bfa567519/go.mod h1:GvvjBRRGRdwPK5ydBHafDWAxML/pGHZbMvKqRZ5+Abc=
-golang.org/x/crypto v0.13.0/go.mod h1:y6Z2r+Rw4iayiXXAIxJIDAJ1zMW4yaTpebo8fPOliYc=
-golang.org/x/crypto v0.19.0/go.mod h1:Iy9bg/ha4yyC70EfRS8jz+B6ybOBKMaSxLj6P6oBDfU=
-golang.org/x/crypto v0.23.0/go.mod h1:CKFgDieR+mRhux2Lsu27y0fO304Db0wZe70UKqHu0v8=
-golang.org/x/crypto v0.31.0/go.mod h1:kDsLvtWBEx7MV9tJOj9bnXsPbxwJQ6csT/x4KIN4Ssk=
-golang.org/x/crypto v0.47.0 h1:V6e3FRj+n4dbpw86FJ8Fv7XVOql7TEwpHapKoMJ/GO8=
-golang.org/x/crypto v0.47.0/go.mod h1:ff3Y9VzzKbwSSEzWqJsJVBnWmRwRSHt/6Op5n9bQc4A=
-golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4/go.mod h1:jJ57K6gSWd91VN4djpZkiMVwK6gcyfeH4XE8wZrZaV4=
-golang.org/x/mod v0.8.0/go.mod h1:iBbtSCu2XBx23ZKBPSOrRkjjQPZFPuis4dIYUhu/chs=
-golang.org/x/mod v0.12.0/go.mod h1:iBbtSCu2XBx23ZKBPSOrRkjjQPZFPuis4dIYUhu/chs=
-golang.org/x/mod v0.15.0/go.mod h1:hTbmBsO62+eylJbnUtE2MGJUyE7QWk4xUqPFrRgJ+7c=
-golang.org/x/mod v0.17.0/go.mod h1:hTbmBsO62+eylJbnUtE2MGJUyE7QWk4xUqPFrRgJ+7c=
-golang.org/x/net v0.0.0-20190620200207-3b0461eec859/go.mod h1:z5CRVTTTmAJ677TzLLGU+0bjPO0LkuOLi4/5GtJWs/s=
-golang.org/x/net v0.0.0-20210226172049-e18ecbb05110/go.mod h1:m0MpNAwzfU5UDzcl9v0D8zg8gWTRqZa9RBIspLL5mdg=
-golang.org/x/net v0.0.0-20220722155237-a158d28d115b/go.mod h1:XRhObCWvk6IyKnWLug+ECip1KBveYUHfp+8e9klMJ9c=
-golang.org/x/net v0.6.0/go.mod h1:2Tu9+aMcznHK/AK1HMvgo6xiTLG5rD5rZLDS+rp2Bjs=
-golang.org/x/net v0.10.0/go.mod h1:0qNGK6F8kojg2nk9dLZ2mShWaEBan6FAoqfSigmmuDg=
-golang.org/x/net v0.15.0/go.mod h1:idbUs1IY1+zTqbi8yxTbhexhEEk5ur9LInksu6HrEpk=
-golang.org/x/net v0.21.0/go.mod h1:bIjVDfnllIU7BJ2DNgfnXvpSvtn8VRwhlsaeUTyUS44=
-golang.org/x/net v0.25.0/go.mod h1:JkAGAh7GEvH74S6FOH42FLoXpXbE/aqXSrIQjXgsiwM=
-golang.org/x/net v0.33.0/go.mod h1:HXLR5J+9DxmrqMwG9qjGCxZ+zKXxBru04zlTvWlWuN4=
-golang.org/x/net v0.48.0 h1:zyQRTTrjc33Lhh0fBgT/H3oZq9WuvRR5gPC70xpDiQU=
-golang.org/x/net v0.48.0/go.mod h1:+ndRgGjkh8FGtu1w1FGbEC31if4VrNVMuKTgcAAnQRY=
-golang.org/x/sync v0.0.0-20190423024810-112230192c58/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
-golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
-golang.org/x/sync v0.1.0/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
-golang.org/x/sync v0.3.0/go.mod h1:FU7BRWz2tNW+3quACPkgCx/L+uEAv1htQ0V83Z9Rj+Y=
-golang.org/x/sync v0.6.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
-golang.org/x/sync v0.7.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
-golang.org/x/sync v0.10.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
-golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
-golang.org/x/sys v0.0.0-20201119102817-f84b799fce68/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
-golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
-golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
-golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
-golang.org/x/sys v0.5.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
-golang.org/x/sys v0.8.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
-golang.org/x/sys v0.12.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
-golang.org/x/sys v0.17.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
-golang.org/x/sys v0.20.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
-golang.org/x/sys v0.28.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
-golang.org/x/telemetry v0.0.0-20240228155512-f48c80bd79b2/go.mod h1:TeRTkGYfJXctD9OcfyVLyj2J3IxLnKwHJR8f4D8a3YE=
-golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1/go.mod h1:bj7SfCRtBDWHUb9snDiAeCFNEtKQo2Wmx5Cou7ajbmo=
-golang.org/x/term v0.0.0-20210927222741-03fcf44c2211/go.mod h1:jbD1KX2456YbFQfuXm/mYQcufACuNUgVhRMnK/tPxf8=
-golang.org/x/term v0.5.0/go.mod h1:jMB1sMXY+tzblOD4FWmEbocvup2/aLOaQEp7JmGp78k=
-golang.org/x/term v0.8.0/go.mod h1:xPskH00ivmX89bAKVGSKKtLOWNx2+17Eiy94tnKShWo=
-golang.org/x/term v0.12.0/go.mod h1:owVbMEjm3cBLCHdkQu9b1opXd4ETQWc3BhuQGKgXgvU=
-golang.org/x/term v0.17.0/go.mod h1:lLRBjIVuehSbZlaOtGMbcMncT+aqLLLmKrsjNrUguwk=
-golang.org/x/term v0.20.0/go.mod h1:8UkIAJTvZgivsXaD6/pH6U9ecQzZ45awqEOzuCvwpFY=
-golang.org/x/term v0.27.0/go.mod h1:iMsnZpn0cago0GOrHO2+Y7u7JPn5AylBrcoWkElMTSM=
-golang.org/x/text v0.3.0/go.mod h1:NqM8EUOU14njkJ3fqMW+pc6Ldnwhi/IjpwHt7yyuwOQ=
-golang.org/x/text v0.3.3/go.mod h1:5Zoc/QRtKVWzQhOtBMvqHzDpF6irO9z98xDceosuGiQ=
-golang.org/x/text v0.3.7/go.mod h1:u+2+/6zg+i71rQMx5EYifcz6MCKuco9NR6JIITiCfzQ=
-golang.org/x/text v0.7.0/go.mod h1:mrYo+phRRbMaCq/xk9113O4dZlRixOauAjOtrjsXDZ8=
-golang.org/x/text v0.9.0/go.mod h1:e1OnstbJyHTd6l/uOt8jFFHp6TRDWZR/bV3emEE/zU8=
-golang.org/x/text v0.13.0/go.mod h1:TvPlkZtksWOMsz7fbANvkp4WM8x/WCo/om8BMLbz+aE=
-golang.org/x/text v0.14.0/go.mod h1:18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=
-golang.org/x/text v0.15.0/go.mod h1:18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=
-golang.org/x/text v0.21.0/go.mod h1:4IBbMaMmOPCJ8SecivzSH54+73PCFmPWxNTLm+vZkEQ=
-golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e/go.mod h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ=
-golang.org/x/tools v0.0.0-20191119224855-298f0cb1881e/go.mod h1:b+2E5dAYhXwXZwtnZ6UAqBI28+e2cm9otk0dWdXHAEo=
-golang.org/x/tools v0.1.12/go.mod h1:hNGJHUnrk76NpqgfD5Aqm5Crs+Hm0VOH/i9J2+nxYbc=
-golang.org/x/tools v0.6.0/go.mod h1:Xwgl3UAJ/d3gWutnCtw505GrjyAbvKui8lOU390QaIU=
-golang.org/x/tools v0.13.0/go.mod h1:HvlwmtVNQAhOuCjW7xxvovg8wbNq7LwfXh/k7wXUl58=
-golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d/go.mod h1:aiJjzUbINMkxbQROHiO6hDPo2LHcIPhhQsa9DLh0yGk=
-golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7/go.mod h1:I/5z698sn9Ka8TeJc9MKroUUfqBBauWjQqLJ2OPfmY0=
+golang.org/x/crypto v0.51.0 h1:IBPXwPfKxY7cWQZ38ZCIRPI50YLeevDLlLnyC5wRGTI=
+golang.org/x/crypto v0.51.0/go.mod h1:8AdwkbraGNABw2kOX6YFPs3WM22XqI4EXEd8g+x7Oc8=
+golang.org/x/net v0.55.0 h1:bcvxaJn3e1U6InsFWt1JUq1aSjnRxLzT2rtD2KfkDF8=
+golang.org/x/net v0.55.0/go.mod h1:L5U2KuzuOe1lY7Z+aWVIKK6qEeJXnXV9yzGA+WCHJww=
 gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
```

## Security Scan

⚠️ **Vulnerabilities found:**
```
=== Symbol Results ===

Vulnerability #1: GO-2026-5039
    Arbitrary inputs are included in errors without any escaping in
    net/textproto
  More info: https://pkg.go.dev/vuln/GO-2026-5039
  Standard library
    Found in: net/textproto@go1.26.2
    Fixed in: net/textproto@go1.26.4
    Example traces found:
      #1: internal/telegram/telegram.go:228:29: telegram.Client.SendDocument calls io.ReadAll, which eventually calls textproto.Reader.ReadMIMEHeader

Vulnerability #2: GO-2026-5037
    Inefficient candidate hostname parsing in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2026-5037
  Standard library
    Found in: crypto/x509@go1.26.2
    Fixed in: crypto/x509@go1.26.4
    Example traces found:
      #1: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls x509.Certificate.Verify
      #2: internal/crypto/crypto.go:87:26: crypto.Encryptor.Encrypt calls io.ReadFull, which eventually calls x509.Certificate.VerifyHostname
      #3: internal/preferences/preferences.go:564:20: preferences.GetWeekKey calls fmt.Sprintf, which eventually calls x509.HostnameError.Error

Vulnerability #3: GO-2026-4971
    Panic in Dial and LookupPort when handling NUL byte on Windows in net
  More info: https://pkg.go.dev/vuln/GO-2026-4971
  Standard library
    Found in: net@go1.26.2
    Fixed in: net@go1.26.3
    Example traces found:
      #1: internal/preferences/gist.go:230:24: preferences.CreateGist calls http.Client.Do, which eventually calls net.Dialer.DialContext

Vulnerability #4: GO-2026-4918
    Infinite loop in HTTP/2 transport when given bad SETTINGS_MAX_FRAME_SIZE in
    net/http/internal/http2 in golang.org/x/net
  More info: https://pkg.go.dev/vuln/GO-2026-4918
  Standard library
    Found in: net/http@go1.26.2
    Fixed in: net/http@go1.26.3
    Example traces found:
      #1: internal/preferences/gist.go:230:24: preferences.CreateGist calls http.Client.Do
      #2: cmd/vga-events-bot/main.go:3018:25: vga.getUpdatesWithTimeout calls http.Client.Get

Your code is affected by 4 vulnerabilities from the Go standard library.
This scan also found 2 vulnerabilities in packages you import and 18
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.
```

## Test Results

✅ All tests pass with updated dependencies
