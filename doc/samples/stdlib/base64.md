
# `base64` module

Base64 encodings (Go's encoding/base64), available without an import as the
`base64` namespace. Each encoding is an `Encoding` value exposing
`EncodeToString(data bytes) <str>` and `DecodeString(s str) <bytes>`.

## Example

```gad
data := bytes("Gad Lang")
base64.StdEncoding.EncodeToString(data)
>>> "R2FkIExhbmc="
base64.RawStdEncoding.EncodeToString(data)
>>> "R2FkIExhbmc"
str(base64.StdEncoding.DecodeString("R2FkIExhbmc="))
>>> "Gad Lang"
```
