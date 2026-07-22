const workflow = require("workflow");
const tasks = require("isolation-fixture-tasks");

module.exports = workflow.compile(workflow.define("isolation-fixture", plan => {
  const source = plan.input("source", {schema: "isolation-source/v1"});
  const transformed = plan.task("transform", tasks.transform({source}), task => task.isolation({
    class: "subprocess.restricted",
    wallTimeMillis: 10000,
    cpuTimeMillis: 5000,
    memoryBytes: 8589934592,
    maxProcesses: 64,
    maxOutputBytes: 1048576,
    maxOutputFiles: 8,
    maxProtocolBytes: 1048576,
  }));
  plan.output("output", transformed.output("output"));
}));
