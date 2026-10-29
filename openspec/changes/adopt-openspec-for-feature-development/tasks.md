# Tasks

## 1. Document the feature-planning workflow

- [ ] 1.1 Add a feature-planning section to `docs/developer.md` describing how contributors capture feature intent and behavior in OpenSpec and include the relevant spec with a feature PR; verify the documented workflow matches the repository's configured `spec-driven` schema and clearly says the spec is optional and not a CI gate.
- [ ] 1.2 Explain that OpenSpec proposal/design material for work affecting the wider Observability group may inform an ADR, while ADR creation remains optional; verify the guidance does not imply automatic ADR creation or a new approval requirement.

## 2. Verify the planning guidance

- [ ] 2.1 Run `openspec validate --change adopt-openspec-for-feature-development` and review the developer-guide text against both `feature-planning` requirements; verify validation passes and the documentation addresses each scenario.

## Workflow follow-up

- Archive the change after implementation and review so the `feature-planning` capability is available as a main OpenSpec spec; verify the archived spec is present.
