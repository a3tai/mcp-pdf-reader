#!/bin/bash

# Release Preparation Script for MCP PDF Reader
# This script handles version tagging, validation, and release coordination

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEFAULT_BRANCH="main"
REMOTE="origin"

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
Release Preparation Script for MCP PDF Reader

USAGE:
    $0 [VERSION] [OPTIONS]

ARGUMENTS:
    VERSION         Version to release (e.g., v0.1.0, v1.2.3-beta.1)
                   If not provided, will suggest next version

OPTIONS:
    --dry-run       Show what would be done without making changes
    --force         Skip some validation checks
    --pre-release   Mark as pre-release (auto-detected from version format)
    --branch BRANCH Source branch for release (default: $DEFAULT_BRANCH)
    --remote REMOTE Git remote to use (default: $REMOTE)
    --help          Show this help message

EXAMPLES:
    # Release next patch version (auto-detected)
    $0

    # Release specific version
    $0 v0.2.0

    # Create pre-release
    $0 v0.2.0-beta.1

    # Dry run to see what would happen
    $0 v0.2.0 --dry-run

    # Force release from different branch
    $0 v0.2.0 --branch develop --force

EOF
}

# Validate git repository state
validate_git_state() {
    log_info "Validating git repository state..."

    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository"
        exit 1
    fi

    # Check if working directory is clean
    if [ -n "$(git status --porcelain)" ]; then
        log_error "Working directory is not clean. Please commit or stash changes."
        git status --short
        exit 1
    fi

    # Check if we're on the correct branch
    local current_branch
    current_branch=$(git branch --show-current)
    if [ "$current_branch" != "$SOURCE_BRANCH" ]; then
        log_error "Not on release branch '$SOURCE_BRANCH'. Currently on '$current_branch'"
        if [ "$FORCE" != "true" ]; then
            log_info "Use --force to override or switch to the correct branch"
            exit 1
        else
            log_warning "Forcing release from '$current_branch' branch"
        fi
    fi

    # Check if branch is up to date with remote
    git fetch "$REMOTE" "$SOURCE_BRANCH" --quiet
    local local_commit remote_commit
    local_commit=$(git rev-parse HEAD)
    remote_commit=$(git rev-parse "$REMOTE/$SOURCE_BRANCH")

    if [ "$local_commit" != "$remote_commit" ]; then
        log_error "Local branch is not up to date with $REMOTE/$SOURCE_BRANCH"
        log_info "Run: git pull $REMOTE $SOURCE_BRANCH"
        exit 1
    fi

    log_success "Git repository state is valid"
}

# Get current version from git tags
get_current_version() {
    local current_tag
    current_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

    if [ -z "$current_tag" ]; then
        echo "v0.0.0"
    else
        echo "$current_tag"
    fi
}

# Suggest next version
suggest_next_version() {
    local current_version="$1"
    local version_without_v="${current_version#v}"

    # Parse semantic version
    if [[ "$version_without_v" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-.*)?$ ]]; then
        local major="${BASH_REMATCH[1]}"
        local minor="${BASH_REMATCH[2]}"
        local patch="${BASH_REMATCH[3]}"
        local prerelease="${BASH_REMATCH[4]}"

        # Suggest patch increment for normal releases
        local next_patch=$((patch + 1))
        echo "v${major}.${minor}.${next_patch}"
    else
        echo "v0.1.0"
    fi
}

# Validate version format
validate_version() {
    local version="$1"

    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        log_error "Invalid version format: $version"
        log_info "Expected format: vX.Y.Z or vX.Y.Z-prerelease"
        log_info "Examples: v1.0.0, v1.2.3-beta.1, v2.0.0-rc.1"
        exit 1
    fi

    # Check if version already exists
    if git tag -l | grep -q "^${version}$"; then
        log_error "Version $version already exists"
        log_info "Existing tags:"
        git tag -l | sort -V | tail -5
        exit 1
    fi
}

# Determine if version is pre-release
is_prerelease() {
    local version="$1"
    if [[ "$version" =~ -(alpha|beta|rc|pre|preview|dev) ]]; then
        echo "true"
    else
        echo "false"
    fi
}

# Run pre-release checks
run_pre_release_checks() {
    log_info "Running pre-release checks..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi

    # Run tests
    log_info "Running tests..."
    if ! make test; then
        log_error "Tests failed"
        exit 1
    fi

    # Run linting if available
    if command -v golangci-lint &> /dev/null; then
        log_info "Running linter..."
        if ! make lint; then
            log_warning "Linting issues found, but continuing..."
        fi
    fi

    # Check if binary can be built
    log_info "Testing build..."
    if ! make build; then
        log_error "Build failed"
        exit 1
    fi

    # Clean up test build
    make clean > /dev/null 2>&1 || true

    log_success "All pre-release checks passed"
}

# Generate preview of release notes
generate_release_notes_preview() {
    local version="$1"
    local previous_tag="$2"

    log_info "Generating release notes preview..."

    if [ -z "$previous_tag" ] || [ "$previous_tag" = "v0.0.0" ]; then
        echo "This will be the initial release!"
        return
    fi

    echo "Changes since $previous_tag:"
    echo "==========================================="

    # Get commits between tags
    local commits
    commits=$(git log --pretty=format:"%s (%h)" --reverse "${previous_tag}"..HEAD)

    if [ -z "$commits" ]; then
        echo "No changes found"
        return
    fi

    # Categorize commits
    local features fixes docs other
    features=""
    fixes=""
    docs=""
    other=""

    while IFS= read -r line; do
        case "$line" in
            feat:*|feature:*|add:*)
                features="$features$line\n"
                ;;
            fix:*|bugfix:*|patch:*)
                fixes="$fixes$line\n"
                ;;
            docs:*|doc:*|documentation:*)
                docs="$docs$line\n"
                ;;
            *)
                other="$other$line\n"
                ;;
        esac
    done <<< "$commits"

    # Display categorized changes
    if [ -n "$features" ]; then
        echo -e "\n🚀 Features:"
        echo -e "$features"
    fi

    if [ -n "$fixes" ]; then
        echo -e "\n🐛 Bug Fixes:"
        echo -e "$fixes"
    fi

    if [ -n "$docs" ]; then
        echo -e "\n📚 Documentation:"
        echo -e "$docs"
    fi

    if [ -n "$other" ]; then
        echo -e "\n🔧 Other Changes:"
        echo -e "$other"
    fi

    echo "==========================================="
}

# Create and push release tag
create_release_tag() {
    local version="$1"
    local is_prerelease="$2"

    log_info "Creating release tag $version..."

    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would create tag: $version"
        log_info "[DRY RUN] Would push tag to $REMOTE"
        return
    fi

    # Create annotated tag
    local tag_message="Release $version"
    if [ "$is_prerelease" = "true" ]; then
        tag_message="Pre-release $version"
    fi

    git tag -a "$version" -m "$tag_message"

    # Push tag to remote
    log_info "Pushing tag to $REMOTE..."
    git push "$REMOTE" "$version"

    log_success "Tag $version created and pushed successfully"
}

# Main release function
prepare_release() {
    local version="$1"
    local current_version previous_tag is_prerelease_flag

    # Get current version if not provided
    if [ -z "$version" ]; then
        current_version=$(get_current_version)
        version=$(suggest_next_version "$current_version")

        log_info "Current version: $current_version"
        log_info "Suggested next version: $version"

        read -p "Use suggested version $version? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            read -p "Enter version (e.g., v1.0.0): " version
        fi
    fi

    # Validate version format
    validate_version "$version"

    # Determine release type
    is_prerelease_flag=$(is_prerelease "$version")
    if [ "$PRE_RELEASE" = "true" ]; then
        is_prerelease_flag="true"
    fi

    # Get previous version for changelog
    previous_tag=$(get_current_version)

    # Display release information
    echo
    log_info "=== RELEASE SUMMARY ==="
    log_info "Version: $version"
    log_info "Type: $([ "$is_prerelease_flag" = "true" ] && echo "Pre-release" || echo "Release")"
    log_info "Branch: $SOURCE_BRANCH"
    log_info "Previous version: $previous_tag"
    log_info "Dry run: $([ "$DRY_RUN" = "true" ] && echo "Yes" || echo "No")"
    echo

    # Generate release notes preview
    generate_release_notes_preview "$version" "$previous_tag"
    echo

    # Confirm release
    if [ "$DRY_RUN" != "true" ]; then
        read -p "Proceed with release? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Release cancelled"
            exit 0
        fi
    fi

    # Validate git state
    validate_git_state

    # Run pre-release checks
    run_pre_release_checks

    # Create and push release tag
    create_release_tag "$version" "$is_prerelease_flag"

    # Final success message
    echo
    log_success "Release $version prepared successfully!"

    if [ "$DRY_RUN" != "true" ]; then
        log_info "GitHub Actions will now build and publish the release"
        log_info "Monitor progress at: https://github.com/$(git config --get remote.$REMOTE.url | sed 's/.*github.com[:/]\([^.]*\).*/\1/')/actions"
        log_info "Release will be available at: https://github.com/$(git config --get remote.$REMOTE.url | sed 's/.*github.com[:/]\([^.]*\).*/\1/')/releases/tag/$version"
    fi
}

# Parse command line arguments
VERSION=""
DRY_RUN="false"
FORCE="false"
PRE_RELEASE="false"
SOURCE_BRANCH="$DEFAULT_BRANCH"

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --force)
            FORCE="true"
            shift
            ;;
        --pre-release)
            PRE_RELEASE="true"
            shift
            ;;
        --branch)
            SOURCE_BRANCH="$2"
            shift 2
            ;;
        --remote)
            REMOTE="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
        *)
            if [ -z "$VERSION" ]; then
                VERSION="$1"
            else
                log_error "Too many arguments"
                show_help
                exit 1
            fi
            shift
            ;;
    esac
done

# Change to project root
cd "$PROJECT_ROOT"

# Run the main function
prepare_release "$VERSION"
