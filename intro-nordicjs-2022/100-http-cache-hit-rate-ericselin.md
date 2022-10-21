👋

Hi, my name is Eric, and I'm here to tell you how I achieved a 100% cache hit ratio using HTTP caching for a highly dynamic site.

⛔

You all know how HTTP caching works - your reverse proxies and your content delivery networks.
You also know that while this works great for static content - marketing websites, images, and javascript files, it works very poorly for dynamic content - basically anything users can write to.
This is because of two problems related to how HTTP caching usually works:

1. Problem number one: cache invalidation is based on time. Once you set the max-age for a response you have to wait for the timer to run out before content is updated.
2. Problem number two: the first request after the timer runs out always requires a round-trip to your server. Slow cold starts are real for HTTP caches.

I'm currently building an order management system for my customer's customers. This is just a simple server that allows reading and writing data. The problem is that the data is stored in HubSpot - a SaaS CRM system - and I need to access this data via the HubSpot API. That means I have lots of network latency, not to mention rate limits that affect response times negatively. The endpoint I'm using the most has a rate limit of 4 requests per second - not great if I want to accomodate more than a few concurrent users.

Because of these problems with HTTP caching, in cases like this we need to look for other options in order to make responses faster and more reliable, right? Maybe mirror the whole dataset in Redis or something? Sure, but solving these problems seems more fun and might lead to a better overall architecture.

🤔

Now, we could use the "purge" functions in our favorite reverse proxy to solve problem one, and hack together a cache warming service to solve problem two. Or do like developers do and re-invent the wheel - and write our own reverse proxy! As expected, the latter is what I did.

🤯

Instead of adding a reverse proxy for static content, and building a massive API-mirroring cache system with CRUD logic and  conflict resolution, I just added one reverse proxy that caches every single read request. (Ok, I had to write the reverse proxy first, but I'd argue that it was still easier.)

This custom reverse proxy does a couple of pretty cool things:

- It manages the cache based on the response headers of writes - i.e. requests that change the underlying data.
- It updates cached responses rather than invalidate them. So when an update is needed, it caches the response before an end user requests it.

**Automatic cache warming.**

So from my app, I can orchestrate the entire cache just by sending along one additional HTTP header in the response after writing data. Everything else just works exactly like a regular reverse proxy. And now I don't have either of the problems we discussed earlier.

I always say: use the platform; and the HTTP protocol is part of the platform for a web developer, whether you like it or not.
I hope I have sparked your interest in using HTTP caching - and indeed the platform - more!
And if you like this idea, please star this repo or email me, to push me to finish cleaning up the code, writing documentation, and open sourcing this bad boy!

*Thank you!*
