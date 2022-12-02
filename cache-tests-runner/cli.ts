import { runTests } from "../../cache-tests/client/runner.mjs";
import tests from "../../cache-tests/tests/index.mjs";
// @deno-types="./types/client/test.d.ts"
import { testResults } from "../../cache-tests/client/test.mjs";
import surrogate from "../../cache-tests/tests/surrogate-control.mjs";
import cacheFetch from "./cache-fetch.mjs";

tests.push(surrogate);
tests.push(cacheFetch);

const [testId] = Deno.args;

const baseUrl = "http:localhost:8080";
const baselineRaw = await Deno.readTextFile("results.json");
const baselineResults: Results = JSON.parse(baselineRaw);

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

Object.entries(testResults).forEach(([key, value]) => {
  const base = baselineResults[key];
  if (testId) {
    console.log(testResults);
    console.log();
  }
  // important to keep track of failed tests!
  if (value !== true) {
    if (base === true) {
      // regression
      console.error("Regression", key, testResults[key]);
    } else if (base === undefined) {
      // new test that is not passing
      console.log("New NOT passing test", key);
    }
    // no change and it is still failing
  } else {
    if (base !== true) {
      // improvement
      console.log("Improvement", key);
    } else if (base === undefined) {
      // new passing test
      console.log("New passing test", key);
    }
    // no change and it is still passing
  }
});
