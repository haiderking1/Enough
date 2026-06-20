// PORT: mirrors backend/enoughhome/home.go
// Ported from backend/enoughhome/home.go

import path from "node:path";

// HomeDir returns the path to the Hollow home directory (default: ~/.hollow).
// Hollow is independent from Enough: it keeps its own config, credentials,
// sessions, skills, and SOUL.md here — separate from ~/.enough.
// If the HOLLOW_HOME environment variable is set, it uses that instead.
export const home_dir = (): string => {
  const eh = process.env.HOLLOW_HOME;
  if (eh !== undefined && eh !== "") {
    return eh;
  }

  const home = process.env.HOME ?? process.env.USERPROFILE;
  if (home === undefined || home === "") {
    return ".hollow";
  }

  return path.join(home, ".hollow");
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
