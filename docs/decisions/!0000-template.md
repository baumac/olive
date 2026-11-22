---
description: The description of the decision.
status: The status of the decision, such as proposed, in progress, or implemented.
---

A decision covers one topic. It describes the current design of the topic.
A change of a requirement updates the decision of its topic. A new topic gets a new decision.
Git keeps the old texts. The decision does not tell its own history.

# Motivation
[motivation]: #motivation

Why are we doing this? What use cases does it support? What is the expected outcome?

# Scope
[scope]: #scope

- What is considered in scope for this decision?
- What is considered out of scope for this decision?
    - List related issues that could be addressed in the future independently of the solution that comes out of this decision.
    - If there are any, provide links to related issues.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

Explain the decision as if it was already included in the repo:
- Explain any new named concepts you are introducing.
- Explain the decision largely in terms of examples.

# Implementation Details
[implementation-details]: #implementation-details

This is the technical portion of the RFC. Explain the design in sufficient detail that:

- Its interaction with other decisions is clear.
- It is reasonably clear how the decision would be implemented.
- Include examples that dissect corner cases.

The section should return to the examples given in the guide-level-explanation section and explain more fully how the detailed proposal makes those examples work.

# Alternatives
[alternatives]: #alternatives

List each design that was rejected or removed. Give one line for each, with the reason.
A removed design keeps its reason here, so that nobody brings it back without that reason.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

List any unresolved questions that still need to be addressed in the decision.