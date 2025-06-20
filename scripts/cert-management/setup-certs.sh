#!/bin/bash

# Certificate Management Script for MCP PDF Reader
# This script helps set up code signing certificates for all platforms

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    cat << EOF
Certificate Management Script for MCP PDF Reader

USAGE:
    $0 [COMMAND] [OPTIONS]

COMMANDS:
    setup-macos         Set up macOS code signing certificates
    setup-windows       Set up Windows code signing certificates
    setup-linux         Set up Linux GPG signing keys
    prepare-secrets     Convert certificates to GitHub Actions secrets format
    validate            Validate existing certificates
    generate-dev        Generate development certificates for testing
    help               Show this help message

OPTIONS:
    --cert-file FILE    Path to certificate file
    --key-file FILE     Path to private key file
    --password PASS     Certificate password
    --output-dir DIR    Output directory for generated files (default: ./certs)

EXAMPLES:
    # Set up macOS certificates
    $0 setup-macos --cert-file developer_id.p12 --password mypassword

    # Set up Windows certificates
    $0 setup-windows --cert-file codesign.pfx --password mypassword

    # Set up Linux GPG signing
    $0 setup-linux --key-file private.key

    # Prepare all certificates for GitHub Actions
    $0 prepare-secrets

    # Validate all certificates
    $0 validate

EOF
}

# Create output directory
create_output_dir() {
    local output_dir="${OUTPUT_DIR:-./certs}"
    mkdir -p "$output_dir"
    chmod 700 "$output_dir"
    echo "$output_dir"
}

# macOS Certificate Setup
setup_macos_certs() {
    log_info "Setting up macOS code signing certificates..."

    local cert_file="$1"
    local password="$2"
    local output_dir
    output_dir=$(create_output_dir)

    if [ ! -f "$cert_file" ]; then
        log_error "Certificate file not found: $cert_file"
        exit 1
    fi

    # Convert to base64 for GitHub Actions
    log_info "Converting certificate to base64..."
    base64 < "$cert_file" > "$output_dir/apple-cert-base64.txt"

    # Test certificate import (temporary keychain)
    log_info "Testing certificate import..."
    local temp_keychain="test-import.keychain"

    security create-keychain -p testpass "$temp_keychain" || true
    security import "$cert_file" -k "$temp_keychain" -P "$password" -T /usr/bin/codesign || {
        log_error "Failed to import certificate. Check password and certificate format."
        security delete-keychain "$temp_keychain" 2>/dev/null || true
        exit 1
    }

    # Get certificate info
    local cert_name
    cert_name=$(security find-identity -v -p codesigning "$temp_keychain" | grep -o '"[^"]*"' | head -1 | tr -d '"')

    security delete-keychain "$temp_keychain"

    if [ -z "$cert_name" ]; then
        log_error "No valid code signing identity found in certificate"
        exit 1
    fi

    log_success "macOS certificate setup complete!"
    log_info "Certificate identity: $cert_name"
    log_info "Base64 certificate saved to: $output_dir/apple-cert-base64.txt"

    # Save GitHub Actions secrets info
    cat > "$output_dir/github-secrets-macos.txt" << EOF
Add these secrets to your GitHub repository:

APPLE_CERT_BASE64: $(cat "$output_dir/apple-cert-base64.txt")
APPLE_CERT_PASSWORD: $password
APPLE_DEVELOPER_ID: $cert_name

Optional (for notarization):
APPLE_ID: your-apple-id@example.com
APPLE_APP_PASSWORD: your-app-specific-password
EOF

    log_info "GitHub Actions secrets template saved to: $output_dir/github-secrets-macos.txt"
}

# Windows Certificate Setup
setup_windows_certs() {
    log_info "Setting up Windows code signing certificates..."

    local cert_file="$1"
    local password="$2"
    local output_dir
    output_dir=$(create_output_dir)

    if [ ! -f "$cert_file" ]; then
        log_error "Certificate file not found: $cert_file"
        exit 1
    fi

    # Convert to base64 for GitHub Actions
    log_info "Converting certificate to base64..."
    base64 < "$cert_file" > "$output_dir/windows-cert-base64.txt"

    # Test certificate (if on Windows/Wine or with openssl)
    log_info "Validating certificate format..."
    if command -v openssl &> /dev/null; then
        openssl pkcs12 -info -in "$cert_file" -passin pass:"$password" -noout || {
            log_error "Failed to validate certificate. Check password and certificate format."
            exit 1
        }
        log_success "Certificate validation successful"
    else
        log_warning "OpenSSL not available, skipping certificate validation"
    fi

    log_success "Windows certificate setup complete!"
    log_info "Base64 certificate saved to: $output_dir/windows-cert-base64.txt"

    # Save GitHub Actions secrets info
    cat > "$output_dir/github-secrets-windows.txt" << EOF
Add these secrets to your GitHub repository:

WINDOWS_CERT_BASE64: $(cat "$output_dir/windows-cert-base64.txt")
WINDOWS_CERT_PASSWORD: $password
EOF

    log_info "GitHub Actions secrets template saved to: $output_dir/github-secrets-windows.txt"
}

# Linux GPG Setup
setup_linux_gpg() {
    log_info "Setting up Linux GPG signing keys..."

    local key_file="$1"
    local passphrase="$2"
    local output_dir
    output_dir=$(create_output_dir)

    if [ -n "$key_file" ] && [ ! -f "$key_file" ]; then
        log_error "Key file not found: $key_file"
        exit 1
    fi

    if [ -z "$key_file" ]; then
        # Generate new GPG key
        log_info "Generating new GPG key for code signing..."

        cat > "$output_dir/gpg-gen-key-config" << EOF
%echo Generating GPG key for MCP PDF Reader code signing
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: MCP PDF Reader
Name-Email: releases@mcp-pdf-reader.example.com
Expire-Date: 2y
Passphrase: ${passphrase:-}
%commit
%echo GPG key generation complete
EOF

        gpg --batch --generate-key "$output_dir/gpg-gen-key-config"
        rm "$output_dir/gpg-gen-key-config"

        # Export the key
        local key_id
        key_id=$(gpg --list-secret-keys --with-colons | grep '^sec:' | cut -d: -f5 | head -1)
        gpg --armor --export-secret-keys "$key_id" > "$output_dir/private-key.asc"
        gpg --armor --export "$key_id" > "$output_dir/public-key.asc"

        log_success "New GPG key generated with ID: $key_id"
        log_info "Private key saved to: $output_dir/private-key.asc"
        log_info "Public key saved to: $output_dir/public-key.asc"

        key_file="$output_dir/private-key.asc"
    else
        # Use existing key file
        log_info "Using existing GPG key file: $key_file"
        cp "$key_file" "$output_dir/private-key.asc"
    fi

    # Save GitHub Actions secrets info
    cat > "$output_dir/github-secrets-linux.txt" << EOF
Add these secrets to your GitHub repository:

GPG_PRIVATE_KEY: $(cat "$output_dir/private-key.asc")
GPG_PASSPHRASE: ${passphrase:-}
EOF

    log_success "Linux GPG setup complete!"
    log_info "GitHub Actions secrets template saved to: $output_dir/github-secrets-linux.txt"
}

# Generate development certificates
generate_dev_certs() {
    log_info "Generating development certificates for testing..."

    local output_dir
    output_dir=$(create_output_dir)

    # Generate self-signed certificate for development
    openssl req -x509 -newkey rsa:4096 -keyout "$output_dir/dev-key.pem" -out "$output_dir/dev-cert.pem" \
        -days 365 -nodes -subj "/CN=MCP PDF Reader Development/O=Development/C=US"

    # Convert to P12 format (for macOS testing)
    openssl pkcs12 -export -out "$output_dir/dev-cert.p12" \
        -inkey "$output_dir/dev-key.pem" -in "$output_dir/dev-cert.pem" \
        -password pass:development

    log_success "Development certificates generated!"
    log_info "Certificate: $output_dir/dev-cert.pem"
    log_info "Private key: $output_dir/dev-key.pem"
    log_info "P12 bundle: $output_dir/dev-cert.p12 (password: development)"

    log_warning "These are self-signed certificates for development only!"
    log_warning "Do not use these for production releases!"
}

# Prepare all secrets for GitHub Actions
prepare_secrets() {
    log_info "Preparing GitHub Actions secrets from existing certificates..."

    local output_dir
    output_dir=$(create_output_dir)

    # Combine all secret templates
    cat > "$output_dir/all-github-secrets.txt" << EOF
# GitHub Actions Secrets for MCP PDF Reader Code Signing
# Add these to your repository settings under Settings > Secrets and variables > Actions

# ================================
# macOS Code Signing Secrets
# ================================
EOF

    if [ -f "$output_dir/github-secrets-macos.txt" ]; then
        cat "$output_dir/github-secrets-macos.txt" >> "$output_dir/all-github-secrets.txt"
    else
        echo "# No macOS certificates found - run: $0 setup-macos" >> "$output_dir/all-github-secrets.txt"
    fi

    cat >> "$output_dir/all-github-secrets.txt" << EOF

# ================================
# Windows Code Signing Secrets
# ================================
EOF

    if [ -f "$output_dir/github-secrets-windows.txt" ]; then
        cat "$output_dir/github-secrets-windows.txt" >> "$output_dir/all-github-secrets.txt"
    else
        echo "# No Windows certificates found - run: $0 setup-windows" >> "$output_dir/all-github-secrets.txt"
    fi

    cat >> "$output_dir/all-github-secrets.txt" << EOF

# ================================
# Linux GPG Signing Secrets
# ================================
EOF

    if [ -f "$output_dir/github-secrets-linux.txt" ]; then
        cat "$output_dir/github-secrets-linux.txt" >> "$output_dir/all-github-secrets.txt"
    else
        echo "# No Linux GPG keys found - run: $0 setup-linux" >> "$output_dir/all-github-secrets.txt"
    fi

    log_success "All GitHub Actions secrets prepared!"
    log_info "Complete secrets file: $output_dir/all-github-secrets.txt"
}

# Validate certificates
validate_certs() {
    log_info "Validating existing certificates..."

    local output_dir="${OUTPUT_DIR:-./certs}"
    local issues=0

    # Check macOS certificates
    if [ -f "$output_dir/apple-cert-base64.txt" ]; then
        log_info "Found macOS certificate base64 file"
        # Could add more validation here
    else
        log_warning "No macOS certificate found"
        ((issues++))
    fi

    # Check Windows certificates
    if [ -f "$output_dir/windows-cert-base64.txt" ]; then
        log_info "Found Windows certificate base64 file"
    else
        log_warning "No Windows certificate found"
        ((issues++))
    fi

    # Check Linux GPG keys
    if [ -f "$output_dir/private-key.asc" ]; then
        log_info "Found Linux GPG private key"
    else
        log_warning "No Linux GPG key found"
        ((issues++))
    fi

    if [ $issues -eq 0 ]; then
        log_success "All certificates appear to be set up correctly!"
    else
        log_warning "Found $issues missing certificate(s)"
        log_info "Run the appropriate setup commands to fix missing certificates"
    fi
}

# Parse command line arguments
COMMAND=""
CERT_FILE=""
KEY_FILE=""
PASSWORD=""
OUTPUT_DIR=""

while [[ $# -gt 0 ]]; do
    case $1 in
        setup-macos|setup-windows|setup-linux|prepare-secrets|validate|generate-dev|help)
            COMMAND="$1"
            shift
            ;;
        --cert-file)
            CERT_FILE="$2"
            shift 2
            ;;
        --key-file)
            KEY_FILE="$2"
            shift 2
            ;;
        --password)
            PASSWORD="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Execute command
case "$COMMAND" in
    setup-macos)
        if [ -z "$CERT_FILE" ]; then
            log_error "--cert-file is required for macOS setup"
            exit 1
        fi
        if [ -z "$PASSWORD" ]; then
            read -s -p "Enter certificate password: " PASSWORD
            echo
        fi
        setup_macos_certs "$CERT_FILE" "$PASSWORD"
        ;;
    setup-windows)
        if [ -z "$CERT_FILE" ]; then
            log_error "--cert-file is required for Windows setup"
            exit 1
        fi
        if [ -z "$PASSWORD" ]; then
            read -s -p "Enter certificate password: " PASSWORD
            echo
        fi
        setup_windows_certs "$CERT_FILE" "$PASSWORD"
        ;;
    setup-linux)
        if [ -z "$PASSWORD" ]; then
            read -s -p "Enter GPG passphrase (or press enter for no passphrase): " PASSWORD
            echo
        fi
        setup_linux_gpg "$KEY_FILE" "$PASSWORD"
        ;;
    prepare-secrets)
        prepare_secrets
        ;;
    validate)
        validate_certs
        ;;
    generate-dev)
        generate_dev_certs
        ;;
    help|"")
        show_help
        ;;
    *)
        log_error "Unknown command: $COMMAND"
        show_help
        exit 1
        ;;
esac
