const baselineResults: Results = JSON.parse(await Deno.readTextFile("results.json"));
const testResults: Results = JSON.parse(await Deno.readTextFile("results-temp.json"));

console.log("\n\n===== RESULTS\n")

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
    } else {
    // no change and it is still failing
      console.log("Failing test", key, testResults[key])
    }
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

console.log("\n=====\n")

if (regressions) {
  console.error("TEST REGRESSIONS ENCOUNTERED")
  Deno.exit(1)
}
