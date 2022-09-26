# Always-Cache

`Always-Cache` is an HTTP cache aiming for 100% cache hit ratio for read requests. This is possible with the following core features:

1. *Pre-cache everything.* On initialization, pre-cache the responses for all possible URLs.
2. *Update cache on demand.* When data changes, update the cached responses for all affected URLs.

Caching behavior (such as content expiry) is controlled via HTTP response headers like any other cache.

## Pre-caching, i.e. cache warming

Traditionally, HTTP caches store responses when requests come in for a particular URL. However, serving cached content means even the first request should be served from cache. `Always-Cache` will therefore cache the entire site or API before any requests come in. You can think of this as Ahead-Of-Time caching instead of Just-In-Time caching. Or simply "pre-caching" or "cache warming".

In order for pre-caching to work, `Always-Cache` needs to know all possible URLs in order to cache them. URLs are collected from the following sources:

- `/sitemap.xml`: Regular XML sitemap for HTML pages, optionally with Google image and video extensions. May also be a sitemap index listing other sitemaps.
- `/sitemap.txt`: Text version of sitemap, with one URL per line.
- Sitemap defined in `/robots.txt`: Either XML or text sitemap per above.
- `/urls.txt`: List of URLs, with one URL per line.

Note that any URLs not listed in the sitemap (which is meant to list HTML pages for search engines) should be included in `urls.txt`. This includes any static assets, such as images, CSS and JS.

## Efficient cache updating

The caching behavior of `Always-Cache` is managed mainly via response headers - just like any other HTTP cache. Cache entries are updated both automatically when the content is about to become stale, and on demand when content is updated (e.g. via a `POST` request).

### Automatic updates of stale content

Before a cached response becomes stale, the cache is updated with a new response. This behavior is described in the HTTP Caching RFC. You can specify your desired caching with the standard `Cache-Control` header or the custom `Always-Cache-Control` header. The advantage of the latter is that it only affects `Always-Cache` and is not sent to the client.

### On-demand updates of updated content

Cached URLs that need to be updated are defined in the `Always-Cache-Update` header. For instance, when a client issues a `POST` request to your backend that updates data shown on `/index.html`, just pass that URL in the header. (Standard HTTP caching does not take into account how to invalidate (i.e. pruge) content from caches, which is a shame.)

## Controlling caching behavior

Standard caching behavior.

## Tips

- Use long max-age -> this is the whole point of `Always-Cache`
- Understand how caching works or use sane defaults
- For dynamic content and data, use the `Always-Cache-Control` header to avoid CDN problems

## Usage

Download source, build, configure, run

## FAQ

What do you mean "read request"?
What do you mean "write request"?
How does pre-caching work?
How does re-caching work?
What is wrong with Redis?
Can I use this in production?
What about authenticated pages?
