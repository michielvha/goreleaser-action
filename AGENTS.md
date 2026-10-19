# Repository instructions

Follow the relevant routes in
https://github.com/vhco-pro/docs/blob/e4d58d2cd69bd42f1efacf91dfdd3f01a24e134a/AGENTS.md.

This isolated next-major branch owns release steps only. CI belongs in
`michielvha/go-ci-action`; versioning uses `michielvha/gitversion-tag-action`
directly. Do not change existing consumer refs or merge this branch without
owner approval. Run `go test ./...` and `shellcheck scripts/*.sh`.
