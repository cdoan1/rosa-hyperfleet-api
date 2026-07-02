# ROSA Hyperfleet API SDK Changelog

All notable changes to the SDK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-07-02

### Added
- Initial SDK release
- Type-safe Go client for ROSA Hyperfleet API
- AWS SigV4 authentication for API Gateway integration
- Support for all platform resources:
  - Cluster management (create, get, list, update, delete)
  - NodePool management (create, get, list, update, delete)
  - Management Cluster operations
  - Account management
  - Authorization (policies, groups, attachments, checks)
  - Trusted Actions (ZOA) operations
- Auto-generated types from OpenAPI specification
- Handwritten client code with:
  - AWS SigV4 signing using `execute-api` service
  - Custom headers (X-Amz-Account-Id, X-Amz-Caller-Arn, X-Amz-User-Id)
  - Context-aware operations
  - Comprehensive error handling with typed helpers
- Working examples:
  - Create cluster
  - List clusters
  - Create ROSA HCP cluster
- Full documentation and usage examples
- SDK build infrastructure:
  - Makefile for code generation and testing
  - oapi-codegen configuration
  - Independent semantic versioning

### Breaking Changes
- N/A (initial release)

---

**Note:** SDK uses independent versioning with `sdk/vX.Y.Z` tag format, allowing the SDK to evolve separately from the API server.
