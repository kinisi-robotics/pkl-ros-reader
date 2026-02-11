# pkl-ros-reader

PKL external module reader for ROS 2 packages.

## Overview

This is a Go-based external reader for the PKL configuration language that resolves `rospkg:///` URIs to ROS package files in both source and installed locations.

## Usage

Register the reader with PKL using the `--external-module-reader` flag to resolve `rospkg:///` URIs in your PKL configuration files.

### URI Format

```pkl
// Generic ROS package format (supports relative imports)
amends "rospkg:///package_name/path/to/file.pkl"
```

The `rospkg:///` scheme maps to ROS package locations using this resolution order:
1. Source directory: `ros/package_name/path/to/file.pkl`
2. Install directory: `$(ros2 pkg prefix package_name)/share/package_name/path/to/file.pkl`

### Examples

```pkl
// Reference a ROS package config file
amends "rospkg:///my_package/config/my_config.pkl"

// Reference a nested config file
amends "rospkg:///my_package/config/system/system_default.pkl"
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

- **Scheme**: `rospkg`
- **Hierarchical URIs**: Yes (supports relative imports)
- **Globbable**: No
- **Local**: Yes (reads from local filesystem and installed packages)

The reader resolves URIs by:
1. Parsing `rospkg:///package_name/path/to/file.pkl` to extract package name and relative path
2. Checking source directory first: `${workspace}/ros/package_name/path/to/file.pkl`
3. If not found, using `ros2 pkg prefix` to locate installed package
4. Reading and returning the file contents as a string

## Integration

The reader is registered with PKL via the `--external-module-reader` flag:

```bash
pkl eval --external-module-reader rospkg=/path/to/pkl-ros-reader config.pkl
```

## License

This project is licensed under the [PolyForm Noncommercial License 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0). You are free to use, modify, and distribute this software for noncommercial purposes only. Commercial use is prohibited without a separate license from the authors.

Copyright (c) 2025 Joshua Smith and Kinisi Robotics.

See [LICENSE](LICENSE) for full terms and [NOTICE](NOTICE) for third-party attributions.
