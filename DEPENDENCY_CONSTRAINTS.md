# Dependency Constraints

This document describes how we manage dependencies for this project.

## Forked OpenShift API (`github.com/openshift/api`)

This project depends on 2 versions of `github.com/openshift/api`:
* The [canonical version](https://github.com/openshift/api).
* A [forked version](https://github.com/rhobs/openshift-api).

**Why Forked:** The observability-operator needs to support both OpenShift console API `v1` and `v1alpha1` for backward compatibility:
- OpenShift >= 4.17 uses `console/v1` API
- OpenShift < 4.17 uses `console/v1alpha1` API

Newer versions of `github.com/openshift/api` (after April 2024) have removed the `console/v1alpha1` API, breaking compatibility with older OpenShift versions. To continue supporting older versions, we forked the library under (https://github.com/rhobs/openshift-api) using the last commit including the `v1alpha1` API and renaming the Go module in `go.mod` to `github.com/rhobs/openshift-api`.

**Impact:** The codebase maintains dual API support with runtime version detection to create the appropriate Console Plugin resources.

**Files Affected:**
- `pkg/controllers/uiplugin/controller.go` - Version detection logic
- `pkg/controllers/uiplugin/components.go` - Dual Console Plugin creation
- `pkg/controllers/uiplugin/plugin_info_builder.go` - Plugin info structure with LegacyProxies
- `pkg/operator/scheme.go` - API scheme registration
- All uiplugin package files using `osv1alpha1` imports

## Updating Dependencies

Dependabot takes care of dependency updates, the configuration is located at `.github/dependabot.yml`.

## Future Considerations

When OpenShift &lt; 4.17 support is no longer needed, we can:
1. Remove `console/v1alpha1` API usage.
2. Remove dual API support code.
3. Remove dependency on `github.com/rhobs/openshift-api` 
4. Update this document 
