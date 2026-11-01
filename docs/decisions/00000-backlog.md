--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

- Add a README.md file to the project for newcomers to understand and get started with the project.
- Implement the Public API OAS in `public-openapi.yaml` in the `api` directory
  - reviews can only be created, updated, and deleted by their author
  - reviews can be read by anyone including unauthenticated users
  - subjects can be created, updated, and deleted by any authenticated user? (TBD on users)
  - subjects can be read by anyone including unauthenticated users
- Generating client libraries and documentation for the Public API
- Design the UI
- Define end-to-end tests in the `tests` directory
- Adding rating aggregate field to subjects
- Adding search