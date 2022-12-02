# Always-Cache

`Always-Cache` is an HTTP cache aiming for 100% cache hit ratio for read requests. This is possible with the following core features:

1. *Update cache instead of invalidating.* When data changes, update the cached responses for all affected URLs.
2. *Pre-cache everything.* On initialization, pre-cache the responses for all possible URLs.

Caching behavior (such as content expiry) is controlled via HTTP response headers like any other cache.

![How always-cache works](how-it-works.png)

## Introducing the idea at the Nordic.JS 2022 conference

[![Nordic.js 2022 • Lightning talk: Eric Selin - Cache all the things for fast and simple backends](http://img.youtube.com/vi/VLAuJO9ivOk/0.jpg)](http://www.youtube.com/watch?v=VLAuJO9ivOk "Nordic.js 2022 • Lightning talk: Eric Selin - Cache all the things for fast and simple backends")

*Click the image to play on YouTube.*

[Slides](intro-nordicjs-2022/100-http-cache-hit-rate-ericselin.pdf) [Transcript-ish](intro-nordicjs-2022/100-http-cache-hit-rate-ericselin.md)

## Revalidate rather than invalidate

The caching behavior of `Always-Cache` follows the HTTP caching standard (RFC 9111) and is managed mainly via response headers - just like any other HTTP cache. Cache entries are updated both automatically when the content is about to become stale, and on demand when content is updated (e.g. via a `POST` request).

### Automatic updates of stale content

Before a cached response becomes stale, the cache is updated with a new response (rather than marking the response stale). When this happens is determined based on [response freshness](https://www.rfc-editor.org/rfc/rfc9111#name-freshness), following the standard.

### On-demand updates of updated content

When content is updated, the cache is updated (rather than invalidated) as is needed. Which URLs to update follows the [standard invalidation rules](https://www.rfc-editor.org/rfc/rfc9111#name-invalidating-stored-respons), and happens when `Always-Cache` receives an ["unsafe request"](https://www.rfc-editor.org/rfc/rfc9110.html#name-common-method-properties). In addition to the standard invalidation methods, it is also possible to use the custom `Cache-Update` HTTP header (see below).

## Caching safe POST requests

The `POST` HTTP method is by definition unsafe. However, in practice, POST requests are oftentimes used only for reading data and are thus "safe". It is, for instance, very common for complicated read operations from a web frontend to use the body of a POST request to transmit query parameters. GraphQL is an entire query language / API that works exclusively on top of POST requests. While requests that change state - unsafe requests - should never be cached, POST requests are in practice not always unsafe in the wild. It would of course be very useful to cache such safe POST requests. This is exactly what `Always-Cache` does.

In order to cache a response to a POST request, set explicit freshness for the response and indicate that this request was safe with the `safe` directive - e.g. `Cache-Control: s-maxage=600; safe`. Then future POST requests with the same body will receive the cached response (given the response is still fresh, of course).

**This behavior technically violates the HTTP caching standard. However, this is with good reason and is applied on an opt-in basis by setting explicit freshness information.**

> Note that at this time, automatic revalidation of POST requests does not work.

## Pre-caching, i.e. cache warming

> Pre-caching is not yet ported to open source version

Traditionally, HTTP caches store responses when requests come in for a particular URL. However, serving only cached content means even the first request should be served from cache. `Always-Cache` will therefore cache the entire site or API before any requests come in. You can think of this as Ahead-Of-Time caching instead of Just-In-Time caching. Or simply "pre-caching" or "cache warming".

In order for pre-caching to work, `Always-Cache` needs to know all possible URLs in order to cache them. URLs are collected from the following sources:

- `/sitemap.xml`: Regular XML sitemap for HTML pages, optionally with Google image and video extensions. May also be a sitemap index listing other sitemaps.
- `/sitemap.txt`: Text version of sitemap, with one URL per line.
- Sitemap defined in `/robots.txt`: Either XML or text sitemap per above.
- `/urls.txt`: List of URLs, with one URL per line.

Note that any URLs not listed in the sitemap (which is meant to list HTML pages for search engines) should be included in `urls.txt`. This includes any static assets, such as images, CSS and JS.

## Controlling caching behavior

`Always-Cache` follows the HTTP caching standard (RFC 9111). However, the following extensions to the standards are defined:

### `safe` extension cache directive

Indicates that the operation at the origin server can be considered "safe" as defined in the HTTP standard. In practice this means that the response may be cached, even if the request was unsafe per the spec. If the response is cached, it MUST NOT be reused unless the body of the incoming request semantically matches the body of the request that led to the stored response.

### `Cache-Fetch` HTTP response header

The `Cache-Fetch` header can be used as an alterative to the invalidation methods explicitly defined in the standard. It allows more flexibility and control around cache updates. Most notably it will always fetch a response for caching, even if no corresponding response already exists in the cache.

#### Syntax

```
cache-fetch-header = "cache-fetch:" SP cache-fetch-string
cache-fetch-string = cache-fetch-path *( ";" SP cache-fetch-attr )
cache-fetch-path   = self / <any CHAR except CTLs or ";">
cache-fetch-attr   = delay-attr
delay-attr         = "delay=" delta-seconds
```

#### Path

The path of the response(s) to revalidate is a URI and may be relative to the current request. The `self` token refers to the URI of the current request.

#### Delay attribute

Delay updating the cache by the specified number of seconds.

## Usage

```
$> always-cache --downstream http://localhost:8081 --port 8080
```

There are many more flags than in the above example. See `always-cache -h` for more information.

## Background

The idea for `always-cache` was - as with many things - born from personal needs. While working with a client, it became pretty much impossible to serve user requests faster than in about one second (yes) without caching. Instead of relying on traditional web app -based caching, HTTP caching was instead used for simplicity. That HTTP caching work became the beginning of this open source solution. See the [introductory talk at Nordic.JS 2022](https://youtu.be/VLAuJO9ivOk).

## Tips

- Use long max-age -> this is the whole point of `Always-Cache`
- Understand how caching works or use sane defaults
- For dynamic content and data, use the `Always-Cache-Control` header to avoid CDN problems

## Contributing

Contributions are always welcome! Whatever they might be. If you star this repo, that is already a big contribution! 😉

The best way to get started is to contact me directly. Please do that!
