# TODO

The current focus is to port the client-specific implementation here, as well as document everything.

- [x] Port basic in-memory cache
- [ ] Implement SQLite cache provider
- [ ] Port cache warmup on startup
    This and the following should use `CacheProvider.OldestExpired()`, one entry at a time.
- [ ] Port cache update on expiry
    In a loop, get the oldest expired entry and update that. If `nil`, sleep for one minute.
- [ ] Port cache update with timeout (`delay` directive)
- [ ] Implement `no-wait` directive
    Remember to add proper testing to header processing.
- [ ] Implement `no-update` directive
- [ ] Implement `no-cache` directive
