# AGENTS.md

Guide for Coding Agents.

## Project Overview

This project allows users to review a variety of things. It is composed of the following surfaces:
- **Public API** for reviewing things with token auth
  - this is implemented and served by the app in the `api` directory
- **Public UI** to interact with the Public API
  - this is implemented in the `ui` directory
- **Pocketbase API** for admin operations with token auth
  - this is served by the app in the `api` directory
- **Pocketbase UI** to interact with the Management API
  - this is served by the app in the `api` directory

## Commands

```bash
make install-tools    # Install dev tools to ./bin/. Needs Go.
make lint             # Run all linters
make lint-spec        # Lint the API specs in specs/api with vacuum
```

TODO update this as more commands exist. Some example commands are:

```bash
make test-api         # Run all API tests
make test-ui          # Run all UI tests
make build            # Build binary to ./bin/gw-admin
make generate         # Generate code and mocks
make run              # Run server (starts Postgres + migrations automatically)
make run-local        # Run with -tags=local for local development mocks
```

## Architecture

- **`makefile`** - the Makefile for building, developing, and testing the project
- **`specs`** - the directory containing the API specifications and the design system
  - **`specs/api/public-openapi.yaml`** - the OpenAPI specification for the Public API
    - the `restaurants` tag has the restaurant and review operations
    - the `users` tag has the sign up, sign in, and token refresh operations
  - **`specs/api/vacuum-ruleset.yaml`** - the vacuum lint rules for the API specifications
  - **`specs/design-system`** - the design system for the Public UI: tokens, component guidelines, logos, and icons
    - Start from `specs/design-system/README.md`
- **`api`** - the directory containining the app that serves the Public API, Pocketbase API, and Pocketbase UI
  - Implements the OpenAPI specification defined in `specs/api/public-openapi.yaml`
  - Written in the Go programming language
  - Built using the Pocketbase framework
- **`ui`** - the directory containing the app that serves the Public UI
  - Implemented against the OpenAPI specification defined in `specs/api/public-openapi.yaml`
  - Written in the TypeScript programming language
  - Built using the Vue framework

## Code Comment Style (Mandatory)

This rule applies to **every** comment you write or touch: Go doc comments, inline
comments, test comments, commit messages, and code blocks in Markdown. It applies
to new comments and to comments you edit for any other reason.

Write comments in **ASD-STE100 Simplified Technical English**.

### Comment Only When Needed

The default is **no comment**. Write a comment only if it gives the reader a fact
that the code cannot show. Valid reasons:

- The code does something unexpected. The comment tells why.
- A constraint, invariant, or side effect is not visible in the code.
- The code is a workaround for an external bug. Include the issue link.
- Go requires a doc comment on an exported identifier.

Do not write a comment that:

- Tells what the code does. The code already shows this.
- Describes your change, for example `added X`, `now uses Y`, `fixed bug`.
  Put this in the commit message.
- Labels an obvious block, for example `// Loop over servers`.
- Tells the steps of the function one by one.

If a better name or a smaller function removes the need for a comment, change the
code. Do not write the comment.

When a comment is necessary, keep only the essential fact. One sentence is the
target. Use more only when the reader needs each sentence.

Do not delete existing comments that you do not touch. This section applies to
comments you write or edit. Commit messages are always necessary.

### Hard Rules

1. **One idea per sentence.** Split any sentence that has two clauses joined by
   `and`, `but`, `so`, `because`, `which`, `where`, or a colon.
2. **Maximum 20 words per sentence.** Count them. If over, split.
3. **One sentence is the target. Maximum 3 sentences per comment block.** If you
   need more, the code needs a better name or a smaller function, not a longer
   comment.
4. **Present tense, active voice.** Write `X changes Y`, not `X is used to
   transform Y` or `Y gets transformed by X`.
5. **One meaning per word.** Use the simplest verb that is correct:
- `change` (not transform, translate, convert, marshal into, coerce)
- `map` only when the code literally maps keys to values
- `get` (not retrieve, fetch, obtain, acquire)
- `send` (not dispatch, propagate, emit, forward on)
- `make` (not construct, instantiate, materialize)
- `use` (not leverage, utilize, employ)
- `start` / `stop` (not initiate, terminate, tear down)
6. **No parenthetical asides and no em dashes.** If the aside matters, make it its
   own sentence. If it does not matter, delete it.
7. **No filler.** Ban list: `simply`, `basically`, `essentially`, `note that`,
   `it is worth noting`, `of course`, `obviously`, `in order to`, `please be aware`.
8. **No hedging narrative.** Do not write `so that an update that leaves it out
   does not silently clear it`. Write the fact: `the stored value stays`.
9. **Do not restate the code.** If a sentence repeats the code or the function
   signature in prose, delete it. If no sentence stays, delete the comment.
10. **Go doc comments start with the identifier name.** `// FooBar does X.`
11. **Wrap at 80 columns.** Never break a line mid-sentence in a way that leaves a
    word glued to the next comment marker.
12. **No emoji, no exclamation marks.**

## Design Decisions

- Substantive design decisions are recorded as decisions under `docs/decisions/`
  Start from `docs/decisions/00000-template.md`.
- Backlog decisions to be completed at a future date are recorded `docs/decisions/backlog/`