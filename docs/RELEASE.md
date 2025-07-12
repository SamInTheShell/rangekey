# Release Process

This document describes the automated release process for RangeDB.

## Automated Release System

RangeDB uses GitHub Actions to automatically build and release binaries for multiple platforms when version tags are pushed to the repository.

### Supported Platforms

The automated build system creates binaries for:

- **Linux x64** (`rangedb-VERSION-linux-amd64.tar.gz`)
- **macOS x64** (`rangedb-VERSION-darwin-amd64.tar.gz`)
- **Windows x64** (`rangedb-VERSION-windows-amd64.zip`)

### Creating a Release

To create a new release:

1. **Tag the release** with a semantic version:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build binaries for all supported platforms
   - Create compressed archives
   - Generate SHA256 checksums
   - Create a GitHub release with all artifacts

### Release Artifacts

Each release includes:

- **Binaries**: Pre-compiled executables for Linux, macOS, and Windows
- **Checksums**: SHA256 checksums for security verification
- **Release Notes**: Comprehensive notes with installation instructions

### Verification

To verify the integrity of downloaded binaries:

```bash
# Download the binary and checksums
wget https://github.com/SamInTheShell/rangekey/releases/download/v1.0.0/rangedb-v1.0.0-linux-amd64.tar.gz
wget https://github.com/SamInTheShell/rangekey/releases/download/v1.0.0/checksums.txt

# Verify the checksum
sha256sum -c checksums.txt
```

### Manual Build

If you prefer to build from source:

```bash
# Clone and build
git clone https://github.com/SamInTheShell/rangekey.git
cd rangekey

# Build for current platform
make build

# Build for all platforms
make build-all

# Create release archives
make release
```

## Workflow Details

### CI Workflow (`.github/workflows/ci.yml`)

Runs on every push and pull request:

- **Test**: Executes all unit tests
- **Build**: Builds binary for current platform
- **Cross-compile**: Verifies cross-compilation for all platforms
- **Artifact Upload**: Uploads build artifacts for review

### Release Workflow (`.github/workflows/release.yml`)

Triggered by version tags:

- **Build**: Cross-compiles binaries for all platforms
- **Archive**: Creates compressed release archives
- **Checksum**: Generates SHA256 checksums
- **Release**: Creates GitHub release with all artifacts

## Release Notes Template

Each release includes automatically generated release notes with:

- Installation instructions for each platform
- Quick start guide
- Binary verification instructions
- Links to full documentation

## Security

All releases include SHA256 checksums for security verification. Always verify the integrity of downloaded binaries using the provided checksums.

## Troubleshooting

### Common Issues

1. **Build Failures**: Check the Actions tab for detailed logs
2. **Missing Platforms**: Ensure all platforms are building successfully
3. **Checksum Mismatches**: Re-download the binary and verify again

### Support

For issues with releases or the automated build system:

1. Check the [GitHub Actions logs](https://github.com/SamInTheShell/rangekey/actions)
2. Create an issue in the [GitHub repository](https://github.com/SamInTheShell/rangekey/issues)
3. Provide detailed information about the platform and error encountered