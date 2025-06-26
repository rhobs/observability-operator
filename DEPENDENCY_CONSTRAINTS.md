# Dependency Constraints

This document describes the constraints and pinned dependencies in this project.

## Pinned Dependencies

### OpenShift API (`github.com/openshift/api`)

**Current Version:** `v0.0.0-20240404200104-96ed2d49b255`

**Why Pinned:** The observability-operator needs to support both OpenShift console API `v1` and `v1alpha1` for backward compatibility:
- OpenShift >= 4.17 uses `console/v1` API  
- OpenShift < 4.17 uses `console/v1alpha1` API

Newer versions of `github.com/openshift/api` (after April 2024) have removed the `console/v1alpha1` API, breaking compatibility with older OpenShift versions.

**Impact:** The codebase maintains dual API support with runtime version detection to create the appropriate Console Plugin resources.

**Files Affected:**
- `pkg/controllers/uiplugin/controller.go` - Version detection logic
- `pkg/controllers/uiplugin/components.go` - Dual Console Plugin creation
- `pkg/controllers/uiplugin/plugin_info_builder.go` - Plugin info structure with LegacyProxies
- `pkg/operator/scheme.go` - API scheme registration
- All uiplugin package files using `osv1alpha1` imports

## Safe to Update Dependencies

The following dependencies can be safely updated:
- Kubernetes API packages (`k8s.io/*`)
- Controller Runtime (`sigs.k8s.io/controller-runtime`)
- Prometheus packages (`github.com/prometheus/*`)
- RHOBS Prometheus Operator (`github.com/rhobs/obo-prometheus-operator`)
- Go standard library extensions (`golang.org/x/*`)
- Utility libraries (`github.com/go-logr/logr`, `github.com/google/go-cmp`, etc.)

## Updating Dependencies

To update dependencies safely:

1. **Individual updates:** Update specific packages excluding openshift/api:
   ```bash
   go get -u k8s.io/api k8s.io/apimachinery k8s.io/client-go
   go get -u sigs.k8s.io/controller-runtime
   go get -u github.com/rhobs/obo-prometheus-operator@v0.83.0-rhobs1
   ```

2. **Avoid bulk updates:** Don't use `go get -u ./...` as it will try to update openshift/api

3. **Always test:** Run `make test-unit` and `make build` after each update

4. **Fix go.sum:** Run `go mod tidy` after updates to fix missing entries

## Future Considerations

When OpenShift < 4.17 support is no longer needed:
1. Remove `console/v1alpha1` API usage
2. Unpin `github.com/openshift/api` 
3. Remove dual API support code
4. Update this document 