import { runTests } from "../../cache-tests/client/runner.mjs";
import tests from "../../cache-tests/tests/index.mjs";
// @deno-types="./types/client/test.d.ts"
import { testResults } from "../../cache-tests/client/test.mjs";

// DISABLE: extra tests
// import surrogate from "../../cache-tests/tests/surrogate-control.mjs";
// import cacheFetch from "./cache-fetch.mjs";

// tests.push(surrogate);
// tests.push(cacheFetch);

const [testId] = Deno.args;

const baseUrl = "http://localhost:8080";

let testsToRun: typeof tests = [];

if (testId) {
  tests.forEach((suite) => {
    suite.tests.forEach((test) => {
      if (test.id === testId) {
        test.dump = true;
        testsToRun = [{
          name: suite.name,
          id: suite.id,
          description: suite.description,
          tests: [test],
        }];
      }
    });
  });
  if (!testsToRun.length) {
    throw new Error(`Cannot find suite ${testId}`);
  }
} else {
  testsToRun = tests;
}

await runTests(testsToRun, fetch, false, baseUrl);

await Deno.writeTextFile(
  "results-temp.json",
  JSON.stringify(testResults, undefined, 2),
);
