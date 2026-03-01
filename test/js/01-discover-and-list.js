var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function repoNames(rows) {
  var names = [];
  var i;
  for (i = 0; i < rows.length; i += 1) {
    names.push(rows[i].name);
  }
  names.sort();
  return names;
}

var manager = wsm.createManager({ defaultJobs: 4 });
var discovered = manager.discover({
  paths: ["."],
  recursive: true,
  maxDepth: 3
});

var repositories = manager.registry.listRepositories({ tags: [] });
assert(discovered.RepositoryCount >= 2, "expected at least 2 repositories discovered");
assert(repositories.length >= 2, "expected listRepositories to return at least 2 rows");

({
  ok: true,
  script: "01-discover-and-list",
  repositoryCountFromDiscover: discovered.RepositoryCount,
  repositoryCountFromList: repositories.length,
  repositories: repoNames(repositories)
});
