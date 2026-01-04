# pdo-tools (Go)

A Go port of [dpethes/pdo-tools](https://github.com/dpethes/pdo-tools), a tool for parsing and exporting Pepakura Designer (PDO) files.

## Features

- Parse PDO files (versions 3, 4, 5, 6).
- Export to SVG (basic implementation).
- CLI tool for easy usage.

## Usage

```bash
# Build
go build ./cmd/pdo-tools

# Export to SVG (default)
./pdo-tools -output output.svg input.pdo

# Export to PDF
./pdo-tools -format pdf -output output.pdf input.pdo

# Dump Textures
./pdo-tools -dump-textures input.pdo
```

## Credits

This project is a port of the original C++/Pascal implementation by [David Pethes](https://github.com/dpethes).

- Original Repository: https://github.com/dpethes/pdo-tools
- Original Author: David Pethes

## License

GPL-2.0 (See [LICENSE](LICENSE))
