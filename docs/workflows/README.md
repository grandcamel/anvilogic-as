# Reference workflow scripts (Waves 1–2)

These are the exact Claude Code `Workflow` scripts that executed Waves 1 and
2, kept as **reference examples** for authoring the Wave 3–5 workflows. They
are not directly re-runnable:

- Absolute paths (project dir, scratchpad) are baked in for the machine that
  ran them — rewrite for yours.
- Lesson learned the hard way: the Workflow `args` parameter can arrive as a
  JSON *string* rather than an object (Wave 1 wrote to a literal `undefined/`
  directory because of this). Wave 2 therefore bakes constants into the
  script — do the same.
- `Date.now()` / `new Date()` are unavailable inside workflow scripts; bake
  the date in as a constant.

Patterns worth copying: parallel fan-out with per-agent StructuredOutput
schemas → synthesis agent → adversarial verifier + completeness critic in
parallel → bounded fix loop (max 2 rounds) keyed on `severity !== 'minor'`;
scaffold agents that must run their own acceptance commands before returning;
verifier prompts that re-fetch cited evidence rather than trusting claims.

| File | Wave | Shape |
|------|------|-------|
| `wave1-capture.reference.js` | 1 | 4 harvesters → synthesize → verify/critic → fix loop |
| `wave2-scaffold.reference.js` | 2 | 2 scaffolders → integration verifier + conventions critic → fix loop |
