# Automated Releases Example

This example demonstrates how RangeDB's automated release system works.

## Example Release Process

### Step 1: Create a Version Tag

```bash
# Create a new version tag
git tag -a v1.0.0 -m "Release v1.0.0 - Initial stable release"

# Push the tag to GitHub
git push origin v1.0.0
```

### Step 2: Automated Build Process

Once the tag is pushed, GitHub Actions automatically:

1. **Builds binaries** for all platforms:
   - Linux x64: `rangedb-v1.0.0-linux-amd64.tar.gz`
   - macOS x64: `rangedb-v1.0.0-darwin-amd64.tar.gz`
   - Windows x64: `rangedb-v1.0.0-windows-amd64.zip`

2. **Generates checksums**:
   ```
   sha256sum *.tar.gz *.zip > checksums.txt
   ```

3. **Creates a GitHub release** with:
   - Release notes with installation instructions
   - Binary attachments
   - Checksum file for verification

### Step 3: User Downloads

Users can then download binaries from the release page:

```bash
# Example: Download Linux binary
wget https://github.com/SamInTheShell/rangekey/releases/download/v1.0.0/rangedb-v1.0.0-linux-amd64.tar.gz
wget https://github.com/SamInTheShell/rangekey/releases/download/v1.0.0/checksums.txt

# Verify integrity
sha256sum -c checksums.txt

# Extract and use
tar -xzf rangedb-v1.0.0-linux-amd64.tar.gz
chmod +x rangedb
./rangedb --help
```

## Release Notes Generated

Each release includes automatically generated notes like:

```markdown
# RangeDB v1.0.0

## What's New

This release includes pre-built binaries for Linux, macOS, and Windows.

## Installation

### Linux (x64)
```bash
wget https://github.com/SamInTheShell/rangekey/releases/download/v1.0.0/rangedb-v1.0.0-linux-amd64.tar.gz
tar -xzf rangedb-v1.0.0-linux-amd64.tar.gz
chmod +x rangedb
./rangedb --help
```

### macOS (x64)
```bash
wget https://github.com/SamInTheShell/rangekey/releases/download/v1.0.0/rangedb-v1.0.0-darwin-amd64.tar.gz
tar -xzf rangedb-v1.0.0-darwin-amd64.tar.gz
chmod +x rangedb
./rangedb --help
```

### Windows (x64)
Download `rangedb-v1.0.0-windows-amd64.zip` and extract the executable.

## Quick Start

```bash
# Start a single node cluster
./rangedb server --cluster-init

# In another terminal, try some operations
./rangedb put /user/123 '{"name": "John", "email": "john@example.com"}'
./rangedb get /user/123
./rangedb range /user/ /user/z
```

## Binary Verification

All binaries are provided with SHA256 checksums. Download `checksums.txt` to verify:

```bash
sha256sum -c checksums.txt
```
```

## Benefits

1. **Automated**: No manual building or uploading required
2. **Consistent**: Same process for every release
3. **Secure**: SHA256 checksums for verification
4. **Multi-platform**: All platforms built simultaneously
5. **Documented**: Complete installation instructions included

## Workflow Files

- `.github/workflows/release.yml`: Handles the automated release process
- `.github/workflows/ci.yml`: Runs tests and builds on every push/PR

This automation ensures that every release is built consistently and includes all necessary files for users to get started quickly.