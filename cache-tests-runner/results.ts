const baselineResults: Results = JSON.parse(
  await Deno.readTextFile("results.json"),
);
const testResults: Results = JSON.parse(
  await Deno.readTextFile("results-temp.json"),
);
// @deno-types="./types/tests.d.ts"
import specs from "../../cache-tests/tests/index.mjs";

console.log("\n\n===== RESULTS\n");

let regressions = 0;
Object.entries(testResults).forEach(([key, value]) => {
  const base = baselineResults[key];
  // important to keep track of failed tests!
  if (value !== true) {
    if (base === true) {
      // regression
      regressions++;
      console.error("Regression", key, testResults[key]);
    } else if (base === undefined) {
      // new test that is not passing
      console.log("New NOT passing test", key);
    } else if (getTest(key)?.kind === undefined) {
      // failing (still) conformance test
      console.log("Failing conformance test", key, testResults[key]);
    }
    // optional test with no change (still failing)
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

console.log("\n=====\n");

if (regressions) {
  console.error("TEST REGRESSIONS ENCOUNTERED");
  Deno.exit(1);
}

function getTest(id: string): Test | undefined {
  return specs.flatMap((s) => s.tests).find((t) => t.id === id);
}
