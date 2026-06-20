"use strict";
// Electron main entry — must run via `electron .`, NOT `bun main.ts`.
// The npm `electron` package is a launcher; app/BrowserWindow exist only inside the Electron process.

const path = require("path");
require("tsx/cjs/api").register({
  tsconfig: path.join(__dirname, "../../tsconfig.json"),
});
require("./main.ts");
