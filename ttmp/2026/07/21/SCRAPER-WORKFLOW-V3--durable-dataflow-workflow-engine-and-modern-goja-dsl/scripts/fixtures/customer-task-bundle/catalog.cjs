"use strict";

module.exports = function registerCustomerTasks(bundle) {
  bundle.task({
    kind: "acme.customer.normalize",
    version: "v1",
    entrypoint: "./tasks/normalize.cjs#run",
    inputSchema: "normalize-customer-input/v1",
    outputs: {dataset: "normalized-customer-dataset-ref/v1"},
    resource: "cpu.transform",
    modules: ["workflow/task", "data/records"],
  });

  bundle.task({
    kind: "acme.customer.validate",
    version: "v1",
    entrypoint: "./tasks/validate.cjs#run",
    inputSchema: "validate-customer-input/v1",
    outputs: {acceptedDataset: "validated-customer-dataset-ref/v1"},
    resource: "cpu.transform",
    modules: ["workflow/task", "data/records"],
  });
};
