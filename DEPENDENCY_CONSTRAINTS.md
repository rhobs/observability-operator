# Dependency Constraints

This document describes how we manage dependencies for this project.

## Forked OpenShift API (`github.com/openshift/api`)

This project depends on 2 versions of `github.com/openshift/api`:
* The [canonical version](https://github.com/openshift/api).
* A [forked version](https://github.com/rhobs/openshift-api).

**Why Forked:** The observability-operator needs to support both OpenShift console API `v1` and `v1alpha1` for backward compatibility:
- OpenShift >= 4.19 uses `console/v1` API (openshift/api)
- OpenShift 4.17 - 4.18 uses `console/v1` API (rhobs/openshift-api)
- OpenShift < 4.17 uses `console/v1alpha1` API (rhobs/openshift-api)

Newer versions of `github.com/openshift/api` (after April 2024) have removed the `console/v1alpha1` API, breaking compatibility with older OpenShift versions. To continue supporting older versions, we forked the library under (https://github.com/rhobs/openshift-api) using the last commit including the `v1alpha1` API and renaming the Go module in `go.mod` to `github.com/rhobs/openshift-api`. The openshift/api `console/v1` adds the `ConsolePlugin.spec.ContentSecurityPolicy` field when marshalling, which is not supported in 4.17 and 4.18, so we utilize rhobs/openshift-api `console/v1` for those versions. 

**Impact:** The codebase maintains dual API support with runtime version detection to create the appropriate Console Plugin resources.

**Files Affected:**
- `pkg/controllers/uiplugin/controller.go` - Version detection logic
- `pkg/controllers/uiplugin/components.go` - Console Plugin version determination and creation
- `pkg/controllers/uiplugin/proxy.go` - Plugin info structure for proxies which are different between v1 and v1alpha1 version
- `pkg/operator/scheme.go` - API scheme registration
- All uiplugin package files using `osv1alpha1` imports

## Updating Dependencies

Dependabot takes care of dependency updates, the configuration is located at `.github/dependabot.yml`.

## Future Considerations

When OpenShift &lt; 4.17 support is no longer needed, we can:
1. Remove `console/v1alpha1` API usage.
4. Update this document

When OpenShift &lt; 4.19 support is no longer needed, we can:
1. Remove dual API support code.
2. Remove dependency on `github.com/rhobs/openshift-api` 
3. Update this document
