# Spec Delta

## Purpose

Defines contributor guidance for using OpenSpec to plan feature work and for optionally carrying broader design discussions into an Observability-group ADR.

## ADDED Requirements

### Requirement: Feature PRs have optional OpenSpec specifications
The repository SHALL document OpenSpec as the recommended way to capture feature intent and behavior, and SHALL state that a relevant spec should accompany a feature PR when practical while remaining optional and not enforced by CI.

#### Scenario: Contributor prepares a feature PR
- **WHEN** a contributor consults the feature-planning guidance
- **THEN** they can find how to create and include an OpenSpec change/spec with the PR
- **AND** they are told that a spec is recommended but not mandatory

### Requirement: OpenSpec planning can inform wider-group ADRs
The repository SHALL explain that OpenSpec planning artifacts for work affecting the wider Observability group may be used as a starting point for an ADR, without requiring or automatically creating one.

#### Scenario: Feature affects the wider Observability group
- **WHEN** contributors identify wider-group impact during feature planning
- **THEN** the guidance indicates that the proposal or design may inform an ADR
- **AND** ADR creation remains optional
