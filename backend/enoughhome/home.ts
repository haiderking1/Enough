// PORT: mirrors backend/enoughhome/home.go
// Ported from backend/enoughhome/home.go

import path from "node:path";

// HomeDir returns the path to the Enough home directory (default: ~/.enough).
// If the ENOUGH_HOME environment variable is set, it uses that instead.
export const home_dir = (): string => {
  const eh = process.env.ENOUGH_HOME;
  if (eh !== undefined && eh !== "") {
    return eh;
  }

  const home = process.env.HOME ?? process.env.USERPROFILE;
  if (home === undefined || home === "") {
    return ".enough";
  }

  return path.join(home, ".enough");
};

/*
PORT STATUS
source path: backend/enoughhome/home.go
source lines: 19
draft lines: 33
confidence: high
status: phase_a_draft
todos:
  - verify process.env + HOME/USERPROFILE fallback matches os.UserHomeDir semantics
  - decide whether to throw on missing home instead of returning ".enough"
notes:
  - No (T, error) return, so Effect was not needed.
*/
