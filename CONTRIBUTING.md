# Contributing to flowbot

Thank you for considering contributing to flowbot! The following guidelines will help you get started and ensure that your contributions are effective and aligned with the project's standards.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Contributing Code](#contributing-code)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Communication](#communication)

## Code of Conduct

Please note that this project is released with a [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project, you agree to abide by its terms.

## How to Contribute

### Reporting Bugs

1. **Search for Existing Issues**: Before reporting a new bug, please search the existing issues to ensure it hasn't already been reported.
2. **Create a New Issue**: If your bug is not already reported, create a new issue with a clear title and detailed description, including steps to reproduce the bug.

### Suggesting Enhancements

1. **Open an Issue**: If you have an idea for an enhancement, open an issue to discuss it with the maintainers.
2. **Provide Details**: Include as much detail as possible about the proposed enhancement, such as use cases and potential benefits.

### Contributing Code

1. **Fork the Repository**: Fork the project repository to your own GitHub account.
2. **Create a Branch**: Create a new branch for your changes, named appropriately (e.g., `feature/new-feature` or `bugfix/issue-123`).
3. **Make Changes**: Implement your changes, adhering to the coding standards below.
4. **Test Your Changes**: Ensure your changes pass all tests and do not introduce new issues.
5. **Commit and Push**: Commit your changes with clear messages and push your branch to your fork.
6. **Open a Pull Request**: Open a pull request against the main branch of the original repository.

## Development Setup

1. **Install Go**: Ensure you have Go installed (version 1.23 or later is recommended).
2. **Clone the Repository**: Clone the repository to your local machine.
   ```sh
   git clone https://github.com/your-username/project-name.git
   cd project-name
   ```
3. **Install Dependencies**: Install any required dependencies.
   ```sh
   go mod tidy
   ```

## Coding Standards

- **Formatting**: Use `gofmt` to format your code.
  ```sh
  go fmt ./...
  ```
- **Linting**: Use `golint` or similar tools to check for style issues.
- **Concurrency**: Follow best practices for concurrency and error handling in Go.

## Testing

- **Write Tests**: Ensure your code is covered by unit tests.
- **Run Tests**: Execute tests using the following command:
  ```sh
  go test ./...
  ```

## Documentation

- **Code Comments**: Document your code with clear comments.
- **README Updates**: If your changes affect the usage or configuration, update the README accordingly.

## Commit Message Guidelines

- **Clear and Concise**: Write clear and concise commit messages.
- **Use the Present Tense**: Start the message with a verb in the present tense (e.g., "Add feature", "Fix bug").

## Pull Request Process

1. **Describe Changes**: Clearly describe the changes in your pull request, including what the problem was and how your changes solve it.
2. **Reference Issues**: If your PR addresses an existing issue, reference it using `Fixes #issue-number`.
3. **Review and Update**: Be prepared to respond to feedback and make necessary updates.

## Communication

- **Issues and Pull Requests**: Use GitHub issues and pull requests for discussions related to specific changes.
- **Slack/Chat**: Join our Slack channel or other chat platforms for real-time discussions (if available).

Thank you for contributing to [Project Name]! Your efforts help make this project better for everyone.
