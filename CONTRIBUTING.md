# Contributing to piStore (piman)

First off, thank you for considering contributing to `piman`! It's people like you who make the open-source community such an amazing place to learn, inspire, and create.

This guide outlines our development workflow and contribution guidelines.

## Code of Conduct

Please be respectful and considerate in all communications. We want to maintain a friendly, welcoming, and inclusive community.

## How Can I Contribute?

### Reporting Bugs
If you find a bug, please open an issue and include:
* A clear description of the problem.
* Steps to reproduce the issue.
* Expected vs. actual behavior.
* Terminal logs or console outputs (especially error messages from `piman serve` or the browser console).

### Suggesting Enhancements
We welcome ideas for new features or user experience improvements. To suggest an enhancement, please open an issue describing:
* The goal of the enhancement.
* How it should work and look (UI suggestions).
* The user value it brings.

### Pull Requests
Ready to submit a code change? Follow these steps:
1. **Fork the Repository** and create your branch from `main`.
2. **Implement Your Changes**:
   * For Go backend changes, make sure the code compiles cleanly and adheres to standard Go formats (`go fmt`).
   * For JS/HTML frontend changes, write clean code and verify that the interface looks cohesive.
3. **Verify Your Code**:
   * Build the project: `go build -o piman main.go`
   * Test the CLI commands and serving capabilities locally.
4. **Submit a PR** with a clear title and description explaining what was changed and why.

## Development Setup

### Prerequisites
* **Go**: version 1.25+ is recommended.
* **Docker & Docker Compose**: installed and configured on the deployment target (e.g. your Raspberry Pi nodes).

### Running Locally
1. Initialize the local configuration directories:
   ```bash
   go run main.go init
   ```
2. Build the binary:
   ```bash
   go build -o piman main.go
   ```
3. Start the local server for development:
   ```bash
   ./piman serve --port 8080
   ```
   Open your browser at `http://localhost:8080` to access the dashboard.
