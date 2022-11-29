# RFC 9111 - HTTP Caching implementation

This module implements the HTTP caching standard (RFC 9111) as a Go module.

Files in this directory correspond to sections in the RFC. The RFC is copied verbatim into comments alongside the implementing code. Text from the RFC is denoted by a paragraph sign, like this:

```
// §  This text here is copied from the RFC verbatim.
```

Filenames of parent (root) sections have a trailing zero added to the section number in order to make them sort before the child sections.
