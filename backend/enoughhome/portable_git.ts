// PORT: mirrors backend/enoughhome/portable_git.go
// Ported from backend/enoughhome/portable_git.go

import path from "node:path";
import { home_dir } from "./home";

// PortableGitDir returns the path where PortableGit is provisioned on Windows.
export const portable_git_dir = (): string => {
  const la = process.env.LOCALAPPDATA;
  if (la !== undefined && la !== "") {
    return path.join(la, "enough", "git");
  }

  return path.join(home_dir(), "git"); // fallback
};

/*
PORT STATUS
source path: backend/enoughhome/portable_git.go
source lines: 14
draft lines: 28
confidence: high
status: phase_a_draft
todos:
  - confirm LOCALAPPDATA behavior on Windows matches Go os.Getenv
  - consider merging portable_git_dir into a single enoughhome module later
notes:
  - No (T, error) return, so Effect was not needed.
*/
