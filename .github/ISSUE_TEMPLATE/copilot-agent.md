---
name: GitHub Copilot Agent Task
about: Template for creating tasks that can be automatically completed by GitHub Copilot Agent
title: '[AGENT] '
labels: ['copilot-agent']
assignees: []

---

## 🤖 GitHub Copilot Agent Task

### Problem/Feature Description
<!-- Describe what needs to be implemented or fixed -->

### Current State
<!-- What's the current behavior? -->

### Expected Outcome
<!-- What should happen after this is complete? -->

### Technical Details
**Files to modify:**
- [ ] `path/to/file1.go`
- [ ] `path/to/file2.go`
- [ ] `path/to/test_file.go`

**Key Requirements:**
- [ ] Requirement 1
- [ ] Requirement 2
- [ ] Requirement 3

### Acceptance Criteria
- [ ] All tests pass
- [ ] Code follows project conventions
- [ ] Documentation updated
- [ ] Integration tests included

### Context
**Related Files:**
- `internal/raft/statemachine.go` - Handles Raft operations
- `internal/server/server.go` - Main server logic
- `internal/partition/manager.go` - Partition management

**Architecture Notes:**
- Uses etcd Raft for consensus
- BadgerDB for storage
- gRPC for communication
- Multi-node cluster support

### Priority
- [ ] High (complete within 24 hours)
- [ ] Medium (complete within 1 week)
- [ ] Low (complete when convenient)

### Additional Notes
<!-- Any specific implementation details, constraints, or preferences -->

---

**Instructions for Agent:**
1. Analyze the current codebase structure
2. Implement the solution following existing patterns
3. Add comprehensive tests
4. Update documentation
5. Ensure all CI checks pass
6. Create a pull request with detailed description

**Agent Configuration:**
- Language: Go
- Framework: etcd Raft, BadgerDB, gRPC
- Testing: Go testing framework
- Style: Follow existing code patterns
