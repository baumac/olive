--- 
This file is not a decision.
Instead, it is a backlog of decisions for the project that still need to be made.
Each sub-bullet point contains notes about the incomplete decision.
---

# Backlog

- Add a README.md file to the project for newcomers to understand and get started with the project.
- Implement the UI based on the Public API Spec
- Add a Content Security Policy to the UI
  - the UI keeps the auth token in `localStorage`. A script that an attacker injects can read it. The CSP stops the browser from running injected scripts.
  - decide the policy when the UI components exist. Some components can need `style-src 'unsafe-inline'`.
  - set the header only on the UI route of decision 00005, not on the PocketBase dashboard. Apply it to the UI build, not to the Vite dev server.
  - first send it as `Content-Security-Policy-Report-Only`, then enforce it
- Define end-to-end tests in the `tests` directory
- Adding rating aggregate field to restaurants
- Adding search
- Adding email verification to sign up
  - sign up gives the same response for new and used email addresses. This removes the account enumeration risk in decision 00001.
  - the API sends a confirmation link for a new email address, or a sign in link for a used one
  - password reset needs the same email setup
