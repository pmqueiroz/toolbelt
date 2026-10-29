# [Short, specific title of what broke]

**Date:** [YYYY-MM-DD]
**Status:** [mitigated / root cause confirmed / open]

## What happened

[The observable symptom, from the reporter's point of view: what they saw,
on what surface, under what conditions. Quote the actual error text if there
was one — verbatim, not paraphrased.]

## What we tried first (wrong track, if applicable)

[Any dead-end investigation, and — important — *why* it seemed reasonable at
the time. Name the lesson explicitly: what assumption turned out to be
wrong, and what should have been checked first instead.]

## What we know

[The confirmed facts: what changed right before the report, what could and
couldn't be reproduced, and what's still circumstantial vs. confirmed. Be
explicit about the difference — don't let a plausible story read as a
confirmed cause.]

## What we did

[The actual fix or mitigation applied, and why it was chosen over other
options — especially if the "correct-looking" fix was passed over for a more
conservative one because the risk of another blind iteration wasn't worth
it.]

## Open question

[If the root cause isn't confirmed: what would falsify the current
mitigation, and what evidence to collect if this recurs.]
