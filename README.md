# Go TemplUI Boilerplate

A modern web application boilerplate built with Go, TemplUI, Templ, and Tailwind CSS CLI.

## Tech Stack

- **Go** - Backend server and application logic
- **TemplUI** - UI component library for Go web applications
- **Templ** - Type-safe HTML templating for Go
- **Tailwind CSS CLI** - Utility-first CSS framework

## Features

- 🚀 Fast development with hot reload
- 🎨 Beautiful UI components with TemplUI
- 🔒 Type-safe HTML templates with Templ
- 💨 Utility-first styling with Tailwind CSS
- 📦 Asset embedding with Go embed
- 🐳 Docker support

## Quick Start

### Prerequisites

- Go 1.21 or later
- Node.js (for Tailwind CSS CLI)

### Installation

1. Clone this repository:
```bash
git clone <your-repo-url>
cd go-templui
```

2. Install Go dependencies:
```bash
go mod tidy
```

3. Install Tailwind CSS CLI:
```bash
npm install -g @tailwindcss/cli
```

4. Build CSS assets:
```bash
make css
```

5. Run the application:
```bash
make run
```

The application will be available at `http://localhost:8080`

## Development

### Available Make Commands

- `make run` - Start the development server
- `make build` - Build the application
- `make css` - Build Tailwind CSS
- `make css-watch` - Watch and rebuild CSS on changes
- `make clean` - Clean build artifacts

### Project Structure

```
├── app/                 # Application logic
├── assets/             # Static assets
│   ├── css/           # CSS files
│   └── assets.go      # Embedded assets
├── tmp/               # Temporary files (development)
├── main.go            # Application entry point
├── Makefile           # Build commands
└── README.md          # This file
```

## Docker

Build and run with Docker:

```bash
docker build -t go-templui .
docker run -p 8080:8080 go-templui
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.