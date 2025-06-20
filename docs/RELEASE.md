# Release Process Documentation

This document describes the complete release process for MCP PDF Reader, including automated workflows, code signing setup, and manual procedures.

## Overview

The release process is designed to be automated and secure, with the following key features:

- **Automated Release Pipeline**: GitHub Actions handles building, signing, and publishing
- **Code Signing**: Binaries are signed for macOS, Windows, and Linux platforms
- **Cross-Platform Builds**: Supports Linux (amd64, arm64), macOS (Intel, Apple Silicon), and Windows (amd64)
- **Release Notes Generation**: Automatic changelog generation from commit history
- **Package Creation**: Ready-to-install packages for all platforms

## Quick Start

### For Regular Releases

```bash
# Prepare and tag a new release
./scripts/prepare-release.sh v1.2.3

# The GitHub Actions workflow will automatically:
# 1. Build binaries for all platforms
# 2. Sign the binaries
# 3. Create release packages
# 4. Generate release notes
# 5. Publish the GitHub release
```

### For Pre-releases

```bash
# Create a pre-release
./scripts/prepare-release.sh v1.2.3-beta.1
```

## Detailed Release Process

### 1. Prerequisites

#### Required Tools
- Go 1.24 or later
- Git
- Make
- Access to GitHub repository with appropriate permissions

#### For Code Signing (Optional but Recommended)
- **macOS**: Apple Developer ID certificate
- **Windows**: Code signing certificate (Authenticode)
- **Linux**: GPG key for signing

### 2. Setting Up Code Signing

#### Initial Setup

Run the certificate management script to set up signing for all platforms:

```bash
./scripts/cert-management/setup-certs.sh help
```

#### macOS Code Signing

```bash
# Set up macOS certificates
./scripts/cert-management/setup-certs.sh setup-macos \
  --cert-file /path/to/developer_id.p12 \
  --password your_cert_password
```

This will:
- Convert your certificate to base64 format
- Generate GitHub Actions secrets template
- Validate the certificate

#### Windows Code Signing

```bash
# Set up Windows certificates
./scripts/cert-management/setup-certs.sh setup-windows \
  --cert-file /path/to/codesign.pfx \
  --password your_cert_password
```

#### Linux GPG Signing

```bash
# Set up Linux GPG signing (generate new key)
./scripts/cert-management/setup-certs.sh setup-linux

# Or use existing key
./scripts/cert-management/setup-certs.sh setup-linux \
  --key-file /path/to/private.key
```

#### GitHub Secrets Configuration

After setting up certificates, add the following secrets to your GitHub repository:

**For macOS:**
- `APPLE_CERT_BASE64`: Base64-encoded P12 certificate
- `APPLE_CERT_PASSWORD`: Certificate password
- `APPLE_DEVELOPER_ID`: Developer ID from certificate
- `APPLE_ID`: (Optional) Apple ID for notarization
- `APPLE_APP_PASSWORD`: (Optional) App-specific password for notarization

**For Windows:**
- `WINDOWS_CERT_BASE64`: Base64-encoded PFX certificate
- `WINDOWS_CERT_PASSWORD`: Certificate password

**For Linux:**
- `GPG_PRIVATE_KEY`: GPG private key in ASCII armor format
- `GPG_PASSPHRASE`: GPG key passphrase

### 3. Release Workflow

#### Automated Release (Recommended)

1. **Prepare the release:**
   ```bash
   ./scripts/prepare-release.sh v1.2.3
   ```

   This script will:
   - Validate the git repository state
   - Run pre-release checks (tests, build)
   - Generate a preview of release notes
   - Create and push the release tag

2. **Monitor GitHub Actions:**
   - The tag push triggers the release workflow automatically
   - Monitor progress at: `https://github.com/your-org/mcp-pdf-reader/actions`

3. **Verify the release:**
   - Check that all binaries were built and signed
   - Verify release notes are accurate
   - Test download links

#### Manual Release Process

If you need to create a release manually:

1. **Create the tag:**
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

2. **Monitor the workflow:**
   - GitHub Actions will automatically trigger on tag push
   - If the workflow fails, you can re-run it from the Actions tab

### 4. Release Types

#### Standard Release
- Format: `v1.2.3`
- Marked as latest release
- Full production release

#### Pre-release
- Format: `v1.2.3-beta.1`, `v1.2.3-rc.1`, `v1.2.3-alpha.1`
- Marked as pre-release in GitHub
- Not marked as latest

### 5. GitHub Actions Workflow Details

The release workflow (`.github/workflows/release.yml`) includes these jobs:

#### `prepare`
- Extracts version information from tag
- Checks for existing releases and deletes them if found
- Generates release notes from commit history

#### `test`
- Runs the complete test suite
- Must pass before proceeding to build

#### `build`
- Builds binaries for all platforms in parallel
- Signs binaries using platform-specific certificates
- Creates checksums (SHA256, SHA512)
- Uploads artifacts

#### `package`
- Downloads all binary artifacts
- Creates installation packages for each platform
- Includes documentation and installation scripts

#### `release`
- Creates the GitHub release
- Uploads all packages and binaries
- Sets appropriate release/pre-release flags

### 6. Version Management

#### Semantic Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (`v2.0.0`): Breaking changes
- **MINOR** (`v1.1.0`): New features, backward compatible
- **PATCH** (`v1.0.1`): Bug fixes, backward compatible

#### Pre-release Versions

- **alpha** (`v1.0.0-alpha.1`): Early development, unstable
- **beta** (`v1.0.0-beta.1`): Feature complete, testing phase
- **rc** (`v1.0.0-rc.1`): Release candidate, final testing

### 7. Post-Release Tasks

1. **Update documentation** if needed
2. **Announce the release** on relevant channels
3. **Monitor for issues** and prepare hotfixes if necessary
4. **Update project planning** for next release

### 8. Troubleshooting

#### Common Issues

**Release workflow fails on code signing:**
- Verify GitHub secrets are correctly set
- Check certificate validity and passwords
- Ensure certificates have proper permissions

**Build fails:**
- Check Go version compatibility
- Verify all dependencies are available
- Run local build to reproduce issue

**Tests fail:**
- Run tests locally: `make test`
- Check for platform-specific issues
- Verify test environment setup

**Tag already exists:**
- Delete the tag locally and remotely:
  ```bash
  git tag -d v1.2.3
  git push origin :refs/tags/v1.2.3
  ```
- Create the tag again with correct version

#### Manual Recovery

If the automated workflow fails and you need to complete the release manually:

1. **Build locally:**
   ```bash
   make build-all
   make package
   ```

2. **Create release manually:**
   - Go to GitHub Releases page
   - Create new release with the tag
   - Upload the packages from `build/releases/`

3. **Sign binaries manually:**
   ```bash
   # macOS
   codesign --sign "Developer ID" --options runtime binary

   # Windows (on Windows with signtool)
   signtool sign /f cert.pfx /p password /t http://timestamp.digicert.com binary.exe

   # Linux
   gpg --detach-sign --armor binary
   ```

### 9. Development and Testing

#### Testing the Release Process

Use dry-run mode to test the release process:

```bash
./scripts/prepare-release.sh v1.2.3 --dry-run
```

#### Development Certificates

Generate self-signed certificates for testing:

```bash
./scripts/cert-management/setup-certs.sh generate-dev
```

**Note:** Development certificates are for testing only and should not be used for production releases.

### 10. Security Considerations

- **Certificate Protection**: Store certificates securely, use GitHub secrets
- **Key Rotation**: Regularly update signing certificates
- **Access Control**: Limit who can create releases
- **Verification**: Always verify signatures after release

### 11. Automation Scripts

#### Available Scripts

- **`scripts/prepare-release.sh`**: Main release preparation script
- **`scripts/cert-management/setup-certs.sh`**: Certificate management
- **`Makefile`**: Build and test commands

#### Script Options

See individual script help:
```bash
./scripts/prepare-release.sh --help
./scripts/cert-management/setup-certs.sh help
```

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Apple Code Signing Guide](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
- [Windows Code Signing](https://docs.microsoft.com/en-us/windows/win32/seccrypto/cryptography-tools)
- [Semantic Versioning](https://semver.org/)
- [GPG Documentation](https://gnupg.org/documentation/)

---

For questions or issues with the release process, please check the troubleshooting section or open an issue in the repository.