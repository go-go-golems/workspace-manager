const wsm = require("wsm");
if (wsm.version !== "0.2.0") throw new Error(`unexpected version ${wsm.version}`);
if (!wsm.consts) throw new Error("missing consts");
if (typeof wsm.createManager !== "function") throw new Error("missing createManager");
console.log("workspace-manager provider ok");
