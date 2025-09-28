# Tag Manager

A generic CLI tool for managing version tags across multiple Go repositories.

## Features

- **Generic Package Discovery**: Automatically discovers Go packages across multiple repositories
- **Custom Tag Naming**: Support for custom tag naming conventions via configuration
- **Interactive Setup**: Guided configuration for new packages with sensible defaults
- **Version Management**: Support for major, minor, and patch version updates
- **Configuration Persistence**: Remembers your tag naming preferences
- **Multi-Repository Support**: Works across any number of Go repositories
- **Git Integration**: Automatic tag creation and pushing

## Installation

1. Navigate to the tag-manager directory:
   ```bash
   cd tag-manager
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the application:
   ```bash
   go build -o tag-manager
   ```

## Usage

### List discovered packages

```bash
./tag-manager list
```

This command will scan for Go modules in the current directory and its subdirectories, displaying all discovered packages.

### Update a package tag

```bash
./tag-manager update
```

The tool will guide you through the entire process interactively:
1. **Package Discovery**: Automatically scan for Go packages in the current directory and its subdirectories
2. **Package Selection**: Choose from the discovered packages
3. **Configuration Setup**: Configure tag naming convention (first time only)
4. **Version Selection**: Choose version type (major/minor/patch)
5. **Confirmation**: Review and confirm the tag update

### Examples

**First-time setup for a package:**
```bash
./tag-manager update
# 1. Select package from discovered list
# 2. Choose tag format (default or custom)
# 3. Select version type
# 4. Confirm tag creation
```

**Subsequent updates:**
```bash
./tag-manager update
# 1. Select package (configuration remembered)
# 2. Select version type
# 3. Confirm tag creation
```

### Configuration

The tool uses a configuration file (`~/.tag-manager.yaml`) to store package-specific settings:

- **Tag Format**: Custom tag naming conventions per package
- **Repository Mapping**: Links packages to their repositories
- **Default Settings**: Global defaults for new packages

**Default Tag Format**: `{package-name}/v{major}.{minor}.{patch}`

**Custom Format Examples**:
- `{package-name}-v{major}.{minor}.{patch}`
- `v{major}.{minor}.{patch}`
- `{package-name}/{major}.{minor}.{patch}`

### Version Types

- `major`: Increments the major version (e.g., v1.2.3 → v2.0.0)
- `minor`: Increments the minor version (e.g., v1.2.3 → v1.3.0)
- `patch`: Increments the patch version (e.g., v1.2.3 → v1.2.4)

## Using Makefile

You can also use the provided Makefile for easier management:

```bash
# Build the application
make build

# Run the tag manager
make run

# Clean build artifacts
make clean
```

**Note**: For the best interactive experience, run the command directly:
```bash
./tag-manager update
```

# Clean build artifacts
make clean
```

## How it works

1. **Package Discovery**: Scans the current directory and its subdirectories for `go.mod` files to discover Go packages
2. **Package Selection**: User selects from discovered packages
3. **Configuration Check**: Checks if package has custom tag format configuration
4. **Interactive Setup**: For new packages, guides user through tag format configuration
5. **Version Selection**: User chooses version type (major/minor/patch)
6. **Tag Calculation**: Calculates new version based on current tag and selected type
7. **Confirmation**: Shows current and new tags for user confirmation
8. **Git Operations**: Creates and pushes the new git tag

## Tag Format

The tool supports flexible tag formats through configuration:

**Default Format**: `{package-name}/v{major}.{minor}.{patch}`

**Examples**:
- `utils/v1.2.3`
- `authorization/v0.1.0`
- `cache/v2.0.0`

**Custom Formats**:
- `utils-v1.2.3` (using `{package-name}-v{major}.{minor}.{patch}`)
- `v1.2.3` (using `v{major}.{minor}.{patch}`)
- `utils/1.2.3` (using `{package-name}/{major}.{minor}.{patch}`)

## Configuration File

The tool creates a configuration file at `~/.tag-manager.yaml` to store your preferences:

```yaml
packages:
  github.com/example/package:
    module_path: github.com/example/package
    tag_format: "{package-name}/v{major}.{minor}.{patch}"
    repository: example-repo
    use_default: true
    last_updated: "2024-01-01T00:00:00Z"
defaults:
  tag_format: "{package-name}/v{major}.{minor}.{patch}"
```
