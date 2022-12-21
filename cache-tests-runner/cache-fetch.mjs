export default {
  name: "Cache-Fetch response header parsing",
  id: "cache-fetch",
  description:
    "These tests check if caches support the Cache-Fetch response header for fetching new responses",
  spec_anchors: [],
  tests: [
    {
      name:
        "Does the cache update a stored response based on the Cache-Fetch header?",
      id: "cache-fetch-update",
      kind: "check",
      requests: [
        {
          // the "root" is created as a directory in order for the relative update to work
          filename: "root/",
          response_headers: [
            ["Cache-Control", "s-maxage=60", false],
          ],
          setup: true,
        },
        {
          filename: "root/updater",
          request_method: "POST",
          response_headers: [
            ["Cache-Update", ".", false],
          ],
          // this affects the client only
          pause_after: true,
        },
        {
          // this is the path for both the server and client
          filename: "root/",
          // server response to send on the third request
          // this should be initiated from the cache immediately after the previous response
          response_headers: [
            ["Cache-Control", "s-maxage=60", false],
          ],
          response_body: "update",
          expected_request_headers_missing: ["test-id"],
          // client request sent after the delay (pause)
          // check that the body is the new body that the server previously sent
          check_body: true,
          // check that this request came from the cache by checking age header
          expected_response_headers: ["Age"],
          // expected_type: "cached" does not work here, since it checks the request count headers
        },
      ],
    },
    {
      name:
        "Does the cache update a stored response based on the Cache-Fetch header?",
      id: "cache-fetch-delay",
      kind: "check",
      requests: [
        {
          // the "root" is created as a directory in order for the relative update to work
          filename: "root/",
          response_headers: [
            ["Cache-Control", "s-maxage=60", false],
          ],
          setup: true,
        },
        {
          filename: "root/updater",
          request_method: "POST",
          response_headers: [
            ["Cache-Update", ".; delay=4", false],
          ],
          pause_after: true,
        },
        {
          filename: "root/",
          // third request should come from cache and be response nr 1
          expected_type: "cached",
          expected_response_headers: [["Server-Request-Count", "1"]],
          // third server response, should happen between client request 3 and 4
          response_headers: [
            ["Cache-Control", "s-maxage=60", false],
          ],
          pause_after: true,
        },
        {
          filename: "root/",
          // request should come from cache and be server response nr 3
          expected_type: "cached",
          expected_response_headers: [["Server-Request-Count", "3"]],
        },
      ],
    },
  ],
};
