#!/usr/bin/env bash

# CODA Build Script
# Builds CODA binaries for multiple platforms

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Project information
PROJECT_NAME="coda"
PROJECT_PATH="github.com/common-creation/coda"
BUILD_DIR="dist"
MAIN_FILE="main.go"

# Get version information
VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u '+%Y-%m-%d %H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

# Build flags
LDFLAGS="-X '${PROJECT_PATH}/cmd.Version=${VERSION}' \
         -X '${PROJECT_PATH}/cmd.Commit=${COMMIT}' \
         -X '${PROJECT_PATH}/cmd.Date=${DATE}' \
         -X '${PROJECT_PATH}/cmd.GoVersion=${GO_VERSION}'"

# Supported platforms
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Function to build for a specific platform
build_platform() {
    local platform=$1
    local os=$(echo "$platform" | cut -d'/' -f1)
    local arch=$(echo "$platform" | cut -d'/' -f2)
    local output="${BUILD_DIR}/${PROJECT_NAME}-${os}-${arch}"
    
    # Add .exe extension for Windows
    if [ "$os" == "windows" ]; then
        output="${output}.exe"
    fi
    
    print_info "Building for ${os}/${arch}..."
    
    GOOS=$os GOARCH=$arch go build \
        -ldflags "${LDFLAGS}" \
        -o "$output" \
        "$MAIN_FILE"
    
    if [ $? -eq 0 ]; then
        print_info "✓ Built: $output"
        # Calculate file size
        local size=$(ls -lh "$output" | awk '{print $5}')
        print_info "  Size: $size"
    else
        print_error "Failed to build for ${os}/${arch}"
        return 1
    fi
}

# Function to build for current platform only
build_current() {
    local output="${BUILD_DIR}/${PROJECT_NAME}"
    
    # Add .exe extension for Windows
    if [ "$OSTYPE" == "msys" ] || [ "$OSTYPE" == "win32" ]; then
        output="${output}.exe"
    fi
    
    print_info "Building for current platform..."
    
    go build \
        -ldflags "${LDFLAGS}" \
        -o "$output" \
        "$MAIN_FILE"
    
    if [ $? -eq 0 ]; then
        print_info "✓ Built: $output"
        local size=$(ls -lh "$output" | awk '{print $5}')
        print_info "  Size: $size"
    else
        print_error "Failed to build"
        return 1
    fi
}

# Function to create release archives
create_archives() {
    print_info "Creating release archives..."
    
    cd "$BUILD_DIR"
    
    for file in *; do
        if [ -f "$file" ]; then
            local archive_name="${file}.tar.gz"
            if [[ "$file" == *.exe ]]; then
                # For Windows, create zip instead
                archive_name="${file%.exe}.zip"
                zip -q "$archive_name" "$file"
            else
                tar -czf "$archive_name" "$file"
            fi
            
            if [ $? -eq 0 ]; then
                print_info "✓ Created: $archive_name"
                rm "$file"  # Remove the original binary
            else
                print_error "Failed to create archive for $file"
            fi
        fi
    done
    
    cd ..
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build CODA binaries for multiple platforms.

OPTIONS:
    -h, --help          Show this help message
    -c, --current       Build for current platform only
    -a, --all           Build for all supported platforms
    -p, --platform OS/ARCH
                        Build for specific platform (e.g., linux/amd64)
    -v, --version VERSION
                        Set version string (default: dev)
    -o, --output DIR    Set output directory (default: dist)
    --archive           Create tar.gz/zip archives after building
    --clean             Clean build directory before building

EXAMPLES:
    # Build for current platform
    $0 --current

    # Build for all platforms
    $0 --all

    # Build specific platform with version
    $0 --platform linux/amd64 --version v1.0.0

    # Build and create archives
    $0 --all --archive
EOF
}

# Parse command line arguments
BUILD_ALL=false
BUILD_CURRENT=false
CREATE_ARCHIVES=false
CLEAN_BUILD=false
SPECIFIC_PLATFORM=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -c|--current)
            BUILD_CURRENT=true
            shift
            ;;
        -a|--all)
            BUILD_ALL=true
            shift
            ;;
        -p|--platform)
            SPECIFIC_PLATFORM="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -o|--output)
            BUILD_DIR="$2"
            shift 2
            ;;
        --archive)
            CREATE_ARCHIVES=true
            shift
            ;;
        --clean)
            CLEAN_BUILD=true
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_info "CODA Build Script"
    print_info "Version: ${VERSION}"
    print_info "Commit: ${COMMIT}"
    print_info "Date: ${DATE}"
    print_info "Go Version: ${GO_VERSION}"
    echo
    
    # Check if go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Clean build directory if requested
    if [ "$CLEAN_BUILD" = true ]; then
        print_info "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
    fi
    
    # Create build directory
    mkdir -p "$BUILD_DIR"
    
    # Download dependencies
    print_info "Downloading dependencies..."
    go mod download
    
    # Run go generate if needed
    if grep -q "//go:generate" ./*.go ./internal/**/*.go ./cmd/**/*.go 2>/dev/null; then
        print_info "Running go generate..."
        go generate ./...
    fi
    
    # Build based on options
    if [ "$BUILD_ALL" = true ]; then
        print_info "Building for all platforms..."
        for platform in "${PLATFORMS[@]}"; do
            build_platform "$platform"
        done
    elif [ "$BUILD_CURRENT" = true ]; then
        build_current
    elif [ -n "$SPECIFIC_PLATFORM" ]; then
        build_platform "$SPECIFIC_PLATFORM"
    else
        # Default to current platform
        build_current
    fi
    
    # Create archives if requested
    if [ "$CREATE_ARCHIVES" = true ]; then
        create_archives
    fi
    
    print_info "Build complete!"
    
    # Show build artifacts
    echo
    print_info "Build artifacts:"
    ls -lh "$BUILD_DIR"
}

# Run main function
main