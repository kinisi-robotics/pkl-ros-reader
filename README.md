# pkl-ros-reader

PKL external module reader for ROS 2 packages.

## Overview

This is a Go-based external reader for the PKL configuration language that resolves `rospkg:///` URIs to ROS package files in both source and installed locations.

## Usage

Register the reader with PKL using the `--external-module-reader` and `--external-resource-reader` flags to resolve `rospkg:///` URIs in your PKL configuration files.

### URI Format

```pkl
// Module import — PKL source files (.pkl)
amends "rospkg:///package_name/path/to/file.pkl"

// Resource read — arbitrary files (YAML, JSON, binary, etc.)
local myYaml = read("rospkg:///package_name/config/params.yaml")
```

The `rospkg:///` scheme maps to ROS package locations using this resolution order:
1. Source directory: `ros/package_name/path/to/file`
2. Install directory: `$(ros2 pkg prefix package_name)/share/package_name/path/to/file`

### Examples

```pkl
// Amend a base config from another ROS package
amends "rospkg:///my_package/config/my_config.pkl"

// Amend a nested config file
amends "rospkg:///my_package/config/system/system_default.pkl"

// Read a YAML file as a resource
local params = read("rospkg:///my_package/config/params.yaml")
```

## Installation

### Via Go Install (Recommended)

```bash
go install github.com/kinisi-robotics/pkl-ros-reader@latest
```

This installs the binary to `~/go/bin/pkl-ros-reader` (ensure `~/go/bin` is in your PATH).

### Building from Source

```bash
git clone https://github.com/kinisi-robotics/pkl-ros-reader.git
cd pkl-ros-reader
go build -o pkl-ros-reader .
```

## Requirements

- Go 1.21 or later
- pkl-go v0.9.0+ (for ExternalReaderClient support)

## Implementation Details

The binary registers two readers under the `rospkg` scheme:

| Reader | Interface | Return type | Pkl usage |
|---|---|---|---|
| Module reader | `ModuleReader` | `string` | `import` / `amends` |
| Resource reader | `ResourceReader` | `[]byte` | `read()` expression |

Both share the same URI resolution logic:
1. Parse `rospkg:///package_name/path/to/file` to extract package name and relative path
2. Check source directory first: `${workspace}/ros/package_name/path/to/file`
3. If not found, use `ros2 pkg prefix` to locate the installed package
4. Read and return the file contents

Common reader properties:
- **Scheme**: `rospkg`
- **Hierarchical URIs**: Yes (supports relative imports)
- **Globbable**: No
- **Local** (module reader only): Yes

## Integration

The reader is registered with PKL via the `--external-module-reader` and `--external-resource-reader` flags. Since a single binary handles both, it is registered twice under the same scheme:

```bash
pkl eval \
  --external-module-reader rospkg=/path/to/pkl-ros-reader \
  --external-resource-reader rospkg=/path/to/pkl-ros-reader \
  config.pkl
```

## License

This project is licensed under the [PolyForm Noncommercial License 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0). You are free to use, modify, and distribute this software for noncommercial purposes only. Commercial use is prohibited without a separate license from the authors.

Copyright (c) 2025 Joshua Smith and Kinisi Robotics.

See [LICENSE](LICENSE) for full terms and [NOTICE](NOTICE) for third-party attributions.
