# MCP PDF Reader

[![Build Status](https://github.com/a3tai/mcp-pdf-reader/workflows/CI/badge.svg)](https://github.com/a3tai/mcp-pdf-reader/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/a3tai/mcp-pdf-reader)](https://goreportcard.com/report/github.com/a3tai/mcp-pdf-reader)
[![Coverage Status](https://coveralls.io/repos/github/a3tai/mcp-pdf-reader/badge.svg?branch=main)](https://coveralls.io/github/a3tai/mcp-pdf-reader?branch=main)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A robust **open source Model Context Protocol (MCP) server** for reading and analyzing PDF documents. This server enables AI assistants and tools to seamlessly interact with PDF files through a standardized protocol.

**🌟 Open Source & Community Driven** - Built with ❤️ by the community, for the community.

## 🚀 Features

- **📄 PDF Processing**: Read, validate, and extract text from PDF documents
- **🔍 Smart Search**: Find PDF files with fuzzy search capabilities
- **📊 Statistics**: Get comprehensive directory and file statistics
- **🔄 Dual Mode Support**:
  - **Stdio Mode**: Standard MCP protocol for AI assistants (Zed, Claude Desktop, etc.)
  - **Server Mode**: HTTP REST API with SSE transport for web integration
- **⚡ Production Ready**: Comprehensive error handling, logging, and graceful shutdown
- **🧪 Well Tested**: 65-76% test coverage with unit and integration tests
- **🛠️ Easy Integration**: Simple installation and configuration

## 🎯 Use Cases

- **AI Code Editors**: Integrate with Zed editor for PDF document analysis
- **Documentation Tools**: Extract and analyze technical documentation
- **Research Assistants**: Process academic papers and research documents
- **Content Management**: Organize and search large PDF collections
- **Web Applications**: HTTP API for web-based PDF processing

## 📦 Installation

### Direct Install (Fastest)

If you have Go installed, you can install directly:

```bash
# Install directly from GitHub
go install github.com/a3tai/mcp-pdf-reader/cmd/mcp-pdf-reader@latest

# Verify installation
mcp-pdf-reader --help
```

### Quick Install (Recommended)

```bash
# Clone the repository
git clone https://github.com/a3tai/mcp-pdf-reader.git
cd mcp-pdf-reader

# Build and install using Go's standard install method
make install

# Ensure Go's bin directory is in your PATH (usually already is)
export PATH="$(go env GOPATH)/bin:$PATH"

# Verify installation
mcp-pdf-reader --help
```

### Manual Build

```bash
# Build from source (creates local binary)
make build

# Or install Go dependencies and build locally
go mod tidy
go build -o mcp-pdf-reader cmd/mcp-pdf-reader/main.go

# Or install directly with Go (installs to GOPATH/bin)
go install github.com/a3tai/mcp-pdf-reader/cmd/mcp-pdf-reader@latest
```

### System Requirements

- **Go 1.21+** for building from source
- **Linux, macOS, or Windows** (tested on all platforms)

## 🖥️ Usage

### MCP Protocol Mode (Default)

Perfect for AI assistants and editors like Zed:

```bash
# Use current directory for PDFs
mcp-pdf-reader

# Specify PDF directory
mcp-pdf-reader -pdfdir=/path/to/documents

# Debug mode
mcp-pdf-reader -pdfdir=/path/to/documents -loglevel=debug
```

### HTTP Server Mode

For web applications and REST API access:

```bash
# Start HTTP server
mcp-pdf-reader -mode=server -pdfdir=/path/to/documents

# Custom host and port
mcp-pdf-reader -mode=server -host=0.0.0.0 -port=9090 -pdfdir=/docs

# Health check
curl http://localhost:8080/health
```

## 🔧 Configuration Options

| Flag | Default | Description |
|------|---------|-------------|
| `-mode` | `stdio` | Server mode: `stdio` or `server` |
| `-pdfdir` | `~/Documents` | Directory containing PDF files |
| `-host` | `127.0.0.1` | Server host (server mode only) |
| `-port` | `8080` | Server port (server mode only) |
| `-loglevel` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `-maxfilesize` | `104857600` | Maximum PDF file size in bytes (100MB) |

## ⚡ Quick Reference

### Common Commands

```bash
# Basic usage (stdio mode for MCP clients)
mcp-pdf-reader -pdfdir=/path/to/pdfs

# Server mode for testing/debugging
mcp-pdf-reader -mode=server -pdfdir=./docs

# Custom port and host
mcp-pdf-reader -mode=server -host=0.0.0.0 -port=9090

# Debug mode
mcp-pdf-reader -mode=server -loglevel=debug -pdfdir=./docs

# Larger file size limit (200MB)
mcp-pdf-reader -maxfilesize=209715200 -pdfdir=./docs
```

### Quick Setup for Popular Editors

| Editor | Config File | Configuration |
|--------|-------------|---------------|
| **Zed** | `~/.config/zed/settings.json` | `"mcp-pdf-reader": {"command": {"path": "mcp-pdf-reader", "args": ["-pdfdir=${workspaceFolder}"]}}` |
| **Cursor** | `~/.cursor/settings.json` | `"mcp-pdf-reader": {"command": "mcp-pdf-reader", "args": ["-pdfdir=${workspaceFolder}"]}` |
| **Claude Desktop** | `~/Library/Application Support/Claude/claude_desktop_config.json` | `"mcp-pdf-reader": {"command": "mcp-pdf-reader", "args": ["-pdfdir=/path/to/docs"]}` |
| **VS Code** | `.vscode/settings.json` | `"claude.mcpServers": {"mcp-pdf-reader": {"command": "mcp-pdf-reader", "args": ["-pdfdir=${workspaceFolder}"]}}` |

### Testing Your Setup

```bash
# 1. Verify installation
mcp-pdf-reader --help

# 2. Test with sample directory
mkdir -p ~/test-pdfs
mcp-pdf-reader -mode=server -pdfdir=~/test-pdfs

# 3. Check health endpoint (server mode)
curl http://localhost:8080/health

# 4. Test MCP tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | mcp-pdf-reader
```

## 📡 MCP Tools

The server provides six main tools via the MCP protocol:

### `pdf_read_file`
Extract text content from a PDF file.

**Parameters:**
- `path` (string): Full path to the PDF file

**Example:**
```json
{
  "path": "/home/user/documents/research.pdf"
}
```

### `pdf_assets_file`
Extract visual assets like images from a PDF file.

**Parameters:**
- `path` (string): Full path to the PDF file

**Example:**
```json
{
  "path": "/home/user/documents/presentation.pdf"
}
```

### `pdf_validate_file`
Validate if a file is a readable PDF.

**Parameters:**
- `path` (string): Full path to the PDF file

**Example:**
```json
{
  "path": "/home/user/documents/document.pdf"
}
```

### `pdf_stats_file`
Get detailed statistics about a PDF file including metadata.

**Parameters:**
- `path` (string): Full path to the PDF file

**Example:**
```json
{
  "path": "/home/user/documents/report.pdf"
}
```

### `pdf_search_directory`
List and search PDF files in a directory with optional fuzzy search.

**Parameters:**
- `directory` (string): Directory path to search
- `query` (string): Optional fuzzy search query

**Example:**
```json
{
  "directory": "/home/user/documents",
  "query": "machine learning"
}
```

### `pdf_stats_directory`
Get statistics about PDF files in a directory.

**Parameters:**
- `directory` (string): Directory path to analyze

**Example:**
```json
{
  "directory": "/home/user/documents"
}
```

## 🎨 Integration Examples

### 🎯 Zed Editor

Add to your Zed settings (`~/.config/zed/settings.json`):

```json
{
  "context_servers": {
    "mcp-pdf-reader": {
      "command": {
        "path": "mcp-pdf-reader",
        "args": ["-pdfdir=${workspaceFolder}"],
        "env": null
      },
      "settings": {}
    }
  }
}
```

**Project-specific Zed configuration** (`.zed/settings.json` in your project):

```json
{
  "context_servers": {
    "mcp-pdf-reader": {
      "command": {
        "path": "mcp-pdf-reader",
        "args": ["-pdfdir=./docs"],
        "env": null
      },
      "settings": {}
    }
  }
}
```

### 🎯 Cursor IDE

Add to your Cursor settings (`~/.cursor/settings.json`):

```json
{
  "mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "${workspaceFolder}"],
      "env": {}
    }
  }
}
```

**For specific PDF directories:**

```json
{
  "mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "/path/to/your/documents"],
      "env": {}
    }
  }
}
```

### 🎯 Windsurf

Add to your Windsurf configuration (`~/.windsurf/settings.json`):

```json
{
  "mcp": {
    "servers": {
      "mcp-pdf-reader": {
        "command": "mcp-pdf-reader",
        "args": ["-pdfdir", "${workspaceRoot}"],
        "env": {}
      }
    }
  }
}
```

**Project-specific Windsurf config** (`.windsurf/settings.json`):

```json
{
  "mcp": {
    "servers": {
      "mcp-pdf-reader": {
        "command": "mcp-pdf-reader",
        "args": ["-pdfdir", "./documentation"],
        "env": {}
      }
    }
  }
}
```

### 🎯 Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "/path/to/your/documents"]
    }
  }
}
```

**For multiple document directories:**

```json
{
  "mcpServers": {
    "mcp-pdf-reader-docs": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "/Users/yourname/Documents"]
    },
    "mcp-pdf-reader-research": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "/Users/yourname/Research/papers"]
    }
  }
}
```

### 🎯 Claude Code (VS Code Extension)

Add to your VS Code settings (`settings.json`):

```json
{
  "claude.mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "${workspaceFolder}"],
      "env": {}
    }
  }
}
```

**Workspace-specific settings** (`.vscode/settings.json`):

```json
{
  "claude.mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "./docs"],
      "env": {}
    }
  }
}
```

### 🎯 Roo Code

Add to your Roo configuration (`~/.roo/config.json`):

```json
{
  "mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "{{workspace}}"],
      "cwd": "{{workspace}}"
    }
  }
}
```

**For specific directories:**

```json
{
  "mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "/path/to/pdfs"],
      "cwd": "/path/to/pdfs"
    }
  }
}
```

### 🎯 Cline (VS Code Extension)

Add to your Cline settings in VS Code (`settings.json`):

```json
{
  "cline.mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "${workspaceFolder}/docs"],
      "env": {}
    }
  }
}
```

**Global Cline configuration:**

```json
{
  "cline.mcpServers": {
    "mcp-pdf-reader": {
      "command": "mcp-pdf-reader",
      "args": ["-pdfdir", "${env:HOME}/Documents"],
      "env": {}
    }
  }
}
```

### 📁 Common Configuration Patterns

#### **Use Current Project Directory**
```bash
# Most editors support workspace variables
-pdfdir=${workspaceFolder}      # Zed, VS Code-based
-pdfdir=${workspaceRoot}        # Windsurf
-pdfdir={{workspace}}           # Roo
```

#### **Use Specific Subdirectory**
```bash
# For documentation in your project
-pdfdir=./docs
-pdfdir=./documentation
-pdfdir=./papers
```

#### **Use Home Directory**
```bash
# For personal document collections
-pdfdir=${env:HOME}/Documents
-pdfdir=/Users/yourname/Documents      # macOS
-pdfdir=/home/yourname/Documents       # Linux
-pdfdir=C:\Users\yourname\Documents    # Windows
```

#### **Multiple Instances**
You can run multiple instances for different directories:

```json
{
  "context_servers": {
    "mcp-pdf-reader-docs": {
      "command": {
        "path": "mcp-pdf-reader",
        "args": ["-pdfdir=./docs", "-port=8080"]
      }
    },
    "mcp-pdf-reader-research": {
      "command": {
        "path": "mcp-pdf-reader",
        "args": ["-pdfdir=/path/to/research", "-port=8081"]
      }
    }
  }
}
```

### 🚀 Quick Setup Tips

1. **After Installation**: The `mcp-pdf-reader` binary will be globally available if `$(go env GOPATH)/bin` is in your PATH (default with Go installations).

2. **Verify Installation**: Run `mcp-pdf-reader --help` to ensure it's working.

3. **Test Configuration**: Start with stdio mode (default) for MCP clients, use server mode for debugging.

4. **Path Variables**: Most editors support workspace variables - use them for portable configurations.

5. **Multiple Directories**: Create separate MCP server instances for different PDF collections.

## 🔧 Troubleshooting

### Installation Issues

#### ❌ Command not found: `mcp-pdf-reader`

**Problem**: After installation, the binary is not found in PATH.

**Solutions**:
```bash
# Check if Go's bin directory is in your PATH
echo $PATH | grep $(go env GOPATH)/bin

# If not found, add to your shell profile
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc  # Linux/WSL
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc   # macOS (if using zsh)

# Reload your shell
source ~/.bashrc  # or ~/.zshrc
```

#### ❌ Permission denied during installation

**Problem**: Installation fails with permission errors.

**Solutions**:
```bash
# Don't use sudo with go install - it should install to your user directory
go install github.com/a3tai/mcp-pdf-reader/cmd/mcp-pdf-reader@latest

# If still having issues, check your GOPATH
go env GOPATH
go env GOBIN
```

#### ❌ Module not found or build errors

**Problem**: Build fails with module or dependency errors.

**Solutions**:
```bash
# Clean module cache and retry
go clean -modcache
go install github.com/a3tai/mcp-pdf-reader/cmd/mcp-pdf-reader@latest

# Or build from source
git clone https://github.com/a3tai/mcp-pdf-reader.git
cd mcp-pdf-reader
go mod tidy
make install
```

### Configuration Issues

#### ❌ MCP server not connecting in editors

**Problem**: Editor can't connect to the MCP server.

**Solutions**:
1. **Verify binary is accessible**:
   ```bash
   which mcp-pdf-reader
   mcp-pdf-reader --help
   ```

2. **Test in stdio mode**:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | mcp-pdf-reader
   ```

3. **Check editor-specific config location**:
   - **Zed**: `~/.config/zed/settings.json`
   - **Cursor**: `~/.cursor/settings.json`
   - **Claude Desktop**: `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
   - **VS Code**: `.vscode/settings.json` (workspace) or user settings

#### ❌ "Directory does not exist" errors

**Problem**: PDF directory path is invalid.

**Solutions**:
```bash
# Use absolute paths
"args": ["-pdfdir=/home/user/Documents"]

# Or verify workspace variables work in your editor
"args": ["-pdfdir=${workspaceFolder}/docs"]

# Create the directory if it doesn't exist
mkdir -p ~/Documents/pdfs
```

#### ❌ "No PDF files found" but files exist

**Problem**: Server can't find PDFs in the specified directory.

**Solutions**:
1. **Check file extensions** (must be `.pdf`):
   ```bash
   ls -la /path/to/pdfs/*.pdf
   ```

2. **Test directory access**:
   ```bash
   mcp-pdf-reader -mode=server -pdfdir=/path/to/pdfs
   # Then visit http://localhost:8080/health
   ```

3. **Check permissions**:
   ```bash
   ls -la /path/to/pdfs/
   # Ensure read permissions on directory and files
   ```

### Runtime Issues

#### ❌ Server crashes or exits immediately

**Problem**: MCP server terminates unexpectedly.

**Solutions**:
1. **Run in server mode for debugging**:
   ```bash
   mcp-pdf-reader -mode=server -pdfdir=./docs -loglevel=debug
   ```

2. **Check for port conflicts** (server mode):
   ```bash
   lsof -i :8080  # Check if port 8080 is in use
   mcp-pdf-reader -mode=server -port=8081  # Try different port
   ```

3. **Verify PDF directory permissions**:
   ```bash
   # Test with a simple directory
   mkdir -p ~/test-pdfs
   mcp-pdf-reader -mode=server -pdfdir=~/test-pdfs
   ```

#### ❌ Large PDF files cause errors

**Problem**: "File too large" or memory errors.

**Solutions**:
```bash
# Increase file size limit (default: 100MB)
mcp-pdf-reader -maxfilesize=209715200  # 200MB

# Check file sizes
ls -lh /path/to/pdfs/*.pdf
```

#### ❌ PDF text extraction fails

**Problem**: PDF content appears empty or garbled.

**Solutions**:
1. **Test with different PDFs** (some PDFs may be image-only or encrypted)
2. **Use validation tool**:
   ```bash
   mcp-pdf-reader -mode=server -pdfdir=./docs
   # Then test with the validate_pdf tool
   ```

### Editor-Specific Issues

#### 🎯 **Zed Editor**
- Restart Zed after config changes
- Check Zed's output panel for MCP errors
- Use absolute paths if workspace variables don't work

#### 🎯 **Cursor IDE**
- Restart Cursor after configuration changes
- Check the "Output" tab for MCP-related logs
- Ensure the MCP extension is enabled

#### 🎯 **Claude Desktop**
- Restart Claude Desktop after config changes
- Check `~/Library/Logs/Claude/` for error logs (macOS)
- Verify JSON syntax in config file

#### 🎯 **VS Code Extensions**
- Check extension logs in the "Output" panel
- Verify the extension supports MCP servers
- Try disabling/re-enabling the extension

### Getting Help

If you're still having issues:

1. **Check the server health** (server mode):
   ```bash
   curl http://localhost:8080/health
   ```

2. **Enable debug logging**:
   ```bash
   mcp-pdf-reader -mode=server -loglevel=debug -pdfdir=./docs
   ```

3. **Create a minimal test case**:
   ```bash
   mkdir test-mcp
   cd test-mcp
   echo "Test content" > test.pdf  # Not a real PDF, but tests basic functionality
   mcp-pdf-reader -mode=server -pdfdir=.
   ```

4. **Open an issue** on [GitHub](https://github.com/a3tai/mcp-pdf-reader/issues) with:
   - Your operating system
   - Go version (`go version`)
   - Editor/tool being used
   - Complete error messages
   - Configuration file contents

## 🧪 Development

### Building and Testing

```bash
# Install dependencies
make deps

# Run tests
make test

# Run tests with coverage
make test-coverage

# Build for development
make build

# Run development server
make run

# Run in server mode
make run-server
```

### Code Quality

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Cross-compile for all platforms
make build-all
```

### Project Structure

```
mcp-pdf-reader/
├── cmd/mcp-pdf-reader/     # Main application entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── mcp/               # MCP server implementation
│   └── pdf/               # PDF processing logic
├── Makefile               # Build and development commands
├── go.mod                 # Go module definition
└── README.md             # This file
```

## 🌐 API Reference (Server Mode)

### Health Check
```http
GET /health
```

Returns server health status and version information.

### MCP Endpoints
```http
GET /sse                   # Server-Sent Events endpoint
POST /message              # MCP message endpoint
```

## 🤝 Contributing

**We love contributions!** This is an open source project and we welcome contributions from everyone. Whether you're fixing bugs, adding features, improving documentation, or helping with tests - every contribution matters.

### How to Contribute

1. **🍴 Fork the repository** on GitHub
2. **🌿 Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **✨ Make your changes** and add comprehensive tests
4. **🧪 Run the test suite**: `make test` (ensure all tests pass)
5. **🎨 Format your code**: `make fmt`
6. **📝 Update documentation** if needed
7. **🚀 Submit a pull request** with a clear description

### Ways to Contribute

- 🐛 **Bug Reports**: Found a bug? Open an issue with reproduction steps
- 💡 **Feature Requests**: Have an idea? We'd love to hear it!
- 📖 **Documentation**: Help improve our docs and examples
- 🧪 **Testing**: Add tests or improve existing ones
- 🔧 **Code**: Fix bugs or implement new features
- 🌍 **Translation**: Help make this accessible to more people

### Development Guidelines

- Write clear, documented code
- Add tests for new functionality
- Follow Go best practices and idioms
- Keep pull requests focused and atomic
- Be respectful and constructive in discussions

## 📊 Performance

- **Memory Efficient**: Streaming PDF processing with configurable limits
- **Fast Search**: Optimized file system traversal and indexing
- **Concurrent Safe**: Handle multiple requests simultaneously
- **Resource Limits**: Configurable file size limits and timeouts

## 🔒 Security

- **Input Validation**: Comprehensive validation of all inputs
- **Path Sanitization**: Prevents directory traversal attacks
- **File Size Limits**: Configurable limits to prevent resource exhaustion
- **Secure Defaults**: Safe configuration out of the box
- **Automated Security Scanning**: Continuous security analysis with gosec

### Security Scanning

This project uses [gosec](https://github.com/securego/gosec) for automated security scanning of Go code. Security scans are automatically run on every pull request and release.

#### Running Security Scans Locally

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
make gosec

# Or run directly with gosec
gosec -conf .gosec.json ./...
```

#### Security Configuration

Security scanning is configured via `.gosec.json` with:
- Customized rules for Go security best practices
- Exclusions for test files and false positives
- Integration with GitHub Security tab via SARIF reports

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🌟 Open Source Community

This project is **proudly open source** and maintained by contributors from around the world. We believe in the power of community-driven development to create better tools for everyone.

### Join Our Community

- **💬 Discussions**: Share ideas and get help in [GitHub Discussions](https://github.com/a3tai/mcp-pdf-reader/discussions)
- **🐛 Issues**: Report bugs or request features in [GitHub Issues](https://github.com/a3tai/mcp-pdf-reader/issues)
- **🎉 Contributors**: Check out our amazing [contributors](https://github.com/a3tai/mcp-pdf-reader/graphs/contributors)

### Project Values

- **🔓 Open**: Transparent development and decision-making
- **🤝 Inclusive**: Welcoming to all contributors regardless of experience level
- **🚀 Quality**: Maintaining high standards through testing and code review
- **📖 Documentation**: Keeping documentation up-to-date and comprehensive

## 🏢 About Rude Company LLC

**Rude Company LLC** is building innovative AI-powered development tools and open source solutions. We create intelligent systems that enhance developer productivity and enable seamless human-AI collaboration.

**A3T** is brought to you by Rude Company LLC and focuses on AI development tools and automation.

- **Website**: [https://rude.la](https://rude.la)
- **A3T Project GitHub**: [https://github.com/a3tai](https://github.com/a3tai)

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/a3tai/mcp-pdf-reader/issues)
- **Discussions**: [GitHub Discussions](https://github.com/a3tai/mcp-pdf-reader/discussions)
- **Support**: For support, please use [GitHub Issues](https://github.com/a3tai/mcp-pdf-reader/issues)

---

Built with ❤️ by Rude Company LLC.
