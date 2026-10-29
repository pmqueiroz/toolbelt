# Incidents

Postmortems for real bugs that made it past review and testing — what broke,
what we actually know (vs. suspected) about why, and what we did about it.
Read this before any migration or destructive operation: some of these are
the reason a "safer" refactor got reverted.

Standing rules earned here (generic starting set — replace/extend with the
project's own once it has real incidents):

- **A crash/error report without a trace pointing at *our* code is not enough
  to diagnose from.** A generic framework/runtime assertion can be thrown far
  from whatever surface the error appears to originate from. Don't theorize a
  fix from a screenshot or a one-line log alone — ask for full output, or
  reproduce locally, before spending another attempt on it.
- **When a reported failure can't be reproduced in the test suite, don't keep
  refining the suspected code.** Revert to the last known-stable
  implementation and re-solve the original problem more conservatively
  instead of iterating blind on unconfirmed code.
- **A shared error string does not mean a shared root cause.** Don't assume
  two failures are the same bug just because the message matches — confirm
  the triggering conditions independently.

Add one entry per incident below, newest first, each linking to its own file
(see [TEMPLATE.md](./TEMPLATE.md) for the shape).
