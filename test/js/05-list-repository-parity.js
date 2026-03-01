var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function names(rows) {
  var out = [];
  var i;
  for (i = 0; i < rows.length; i += 1) {
    out.push(rows[i].name);
  }
  out.sort();
  return out;
}

function sameList(a, b) {
  if (a.length !== b.length) {
    return false;
  }
  var i;
  for (i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
}

var manager = wsm.createManager({ defaultJobs: 4 });
var flat = manager.listRepositories({ tags: [] });
var grouped = manager.registry.listRepositories({ tags: [] });

var flatNames = names(flat);
var groupedNames = names(grouped);

assert(flat.length >= 2, "expected at least two repositories from listRepositories");
assert(sameList(flatNames, groupedNames), "flat and grouped repository listing should match");

({
  ok: true,
  script: "05-list-repository-parity",
  repositoryCount: flat.length,
  repositoryNames: flatNames,
  parity: sameList(flatNames, groupedNames)
});
