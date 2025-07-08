# Bash commands
- go test -run=XXX_SHOULD_NEVER_MATCH_XXX ./test/e2e: Compile e2e tests

# Project structure
- ./test/e2e: Contains end-to-end tests for the project
- ./deploy/crds/common: Contains common custom resource definitions (CRDs) used in the project
- ./pkg/apis: Contains API definitions for the project including custom resource definitions (CRDs)