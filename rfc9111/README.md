# RFC 9111 - HTTP Caching implementation

This module implements the HTTP caching standard (RFC 9111) as a Go module.

Files in this directory correspond to sections in the RFC. The RFC is copied verbatim into comments alongside the implementing code. Text from the RFC is denoted by a paragraph sign, like this:

```
// §  This text here is copied from the RFC verbatim.
```

Unfortunately the files (file names in e.g. `ls`) are sorted a bit weird, with the parent section sorted after the subsections. This is because unicode sorting ignores punctuation.