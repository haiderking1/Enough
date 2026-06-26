// Pure function on the session manager — no Agent import, no side effects.

// sessionFingerprints adapts the session store to the evidence seeder.
export function sessionFingerprints(sm: any): any[] {
  const out: any[] = [];
  if (sm && typeof sm.fingerprints === "function") {
    const list = sm.fingerprints().list();
    for (const fp of list) {
      out.push({ path: fp.path, after_hash: fp.after_hash });
    }
  }
  return out;
}
