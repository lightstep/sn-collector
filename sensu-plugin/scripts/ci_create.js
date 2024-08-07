/**
 * ci_create.js
 *
 * Creates Service CIs and relationships based on metric metadata.
 *
 * Check input:
 * [
 *        {
 *           'client':'...',
 *           'agent_id':'...',
 *           'check':{
 *              'command':'...',
 *              'name':'...',
 *              'interval':60,
 *              'timeout':60,
 *              'output':'...see below. ... ',
 *              'status':'..',
 *           }
 *        }
 *     ]
 *
 * Example check output:
 * lima-default.system.cpu.load.average.15m 0.33 1722967059
 * lima-default.system.cpu.time;cpu=cpu0;state=system 8163.79 1722967059
 * traces-service.graph.request.total;client=telemetrygen;connection_type=virtual_node;failed=false;server=telemetrygen-server 22 1722965304
 */

function parseLine(line) {
  // Regular expression to capture metric name, labels, value, and timestamp
  var regex = /^([^ ;]+)(;[^ ]+)? ([^ ]+) ([^ ]+)$/;
  var match = line.match(regex);

  if (match) {
    var metricName = match[1];
    var labelsString = match[2];
    var value = match[3];
    var timestamp = match[4];

    // Parse labels into an object
    var labels = {};
    if (labelsString) {
      // Remove leading semicolon and split the labels by semicolon
      labelsString = labelsString.substring(1);
      var labelsArray = labelsString.split(";");
      labelsArray.forEach(function (label) {
        var parts = label.split("=");
        labels[parts[0]] = parts[1];
      });
    }

    return {
      metricName: metricName,
      labels: labels,
      value: value,
      timestamp: timestamp,
    };
  }

  return null;
}

function createServicePayloadItem(serviceName, metricName) {
  return {
    className: "cmdb_ci_service_discovered",
    values: {
      name: serviceName,
      sys_class_name: "cmdb_ci_service_discovered",
      short_description: "Created based on metric: " + metricName,
    },
  };
}

function processCheckOutput(output) {
  var lines = output.split("\n");
  var results = lines.map(parseLine);

  var irePayload = {
    items: [],
    relations: [],
  };

  var foundServices = {};
  var foundRelations = {};

  function addPayloadItem(serviceName, metricName) {
    if (foundServices[serviceName]) {
      return;
    }

    irePayload.items.push(createServicePayloadItem(serviceName, metricName));
    foundServices[serviceName] = true;
  }

  function addPayloadRelation(client, server) {
    if (foundRelations[client + server]) {
      return;
    }

    if (client == server) {
      return;
    }

    var clientIndex = irePayload.items.findIndex(function (item) {
      return item.values.name === client;
    });
    var serverIndex = irePayload.items.findIndex(function (item) {
      return item.values.name === server;
    });

    if (clientIndex === -1 || serverIndex === -1) {
      return;
    }
    irePayload.relations.push({
      parent: clientIndex,
      child: serverIndex,
      type: "Depends on::Used by",
    });
    foundRelations[client + server] = true;
  }

  // Process each metric and create or update the CI if necessary
  results.forEach(function (metric) {
    if (!metric || !metric.labels) {
      return;
    }

    if (metric.labels.client) {
      addPayloadItem(metric.labels.client, metric.metricName);
    }

    if (metric.labels.server) {
      addPayloadItem(metric.labels.server, metric.metricName);
    }

    if (metric.labels.client && metric.labels.server) {
      addPayloadRelation(metric.labels.client, metric.labels.server);
    }
  });

  gs.info(
    "[OpenTelemetry Discovery] IRE payload: " + JSON.stringify(irePayload)
  );
  var result = sn_cmdb.IdentificationEngine.createOrUpdateCI(
    "ServiceNow",
    JSON.stringify(irePayload)
  );
  gs.info("[OpenTelemetry Discovery] IRE result: " + JSON.stringify(result));
}

// Uncomment for testing purposes
// var checkResults = [
//    {
//        check: {
//            output: "traces-service.graph.request.total;client=foo;connection_type=virtual_node;failed=false;server=bar-server 22 1722965304\ntraces-service.graph.request.total;client=foo;connection_type=virtual_node;failed=false;server=baz-server 22 1722965304"
//        }
//    }
// ]

gs.info("[OpenTelemetry Discovery] processing checks...");
gs.info("" + JSON.stringify(checkResults));

try {
  for (var index = 0; index < checkResults.length; index++) {
    var check = checkResults[index].check;
    processCheckOutput(check.output);
  }
} catch (e) {
  gs.error("[OpenTelemetry Discovery] Error processing checks: " + e);
  gs.error(e.stack);
}
