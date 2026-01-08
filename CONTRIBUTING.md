# Contributing to witr

First off, thank you for considering contributing to **witr**! It's people like you that make the open-source community such an amazing place to learn, inspire, and create.

All types of contributions are encouraged and valued. See the [Table of Contents](#table-of-contents) for different ways to help and details about how this project handles them. Please make sure to read the relevant section before making your contribution. It will make it a lot easier for us maintainers and smooth out the experience for all involved. The community looks forward to your contributions. 🎉

> If you like the project, but just don't have time to contribute, that's fine. There are other easy ways to support the project and show your appreciation, which we would also be very happy about:
> - Star the project
> - Tweet about it
> - Refer this project in your project's readme
> - Mention the project at local meetups and tell your friends/colleagues

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [I Have a Question](#i-have-a-question)
- [I Want To Contribute](#i-want-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Your First Code Contribution](#your-first-code-contribution)
  - [Improving Documentation](#improving-documentation)
- [Styleguides](#styleguides)
  - [Commit Messages](#commit-messages)
- [Join The Project Team](#join-the-project-team)

## Building from source

When you need to verify a change locally, compile the CLI with Go 1.25+
so that the embedded version data stays accurate:

```bash
git clone https://github.com/pranshuparmar/witr.git
cd witr
go build -ldflags "-X main.version=v0.0.0-dev -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date +%Y-%m-%d)'" -o witr ./cmd/witr
./witr --help  # quick smoke test
```

- The `-ldflags` block injects commit/date metadata for `witr --version`.
- The resulting `witr` binary lands in the repo root.

## Code Style
## Code of Conduct

This project and everyone participating in it is governed by the [witr Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to .

## I Have a Question

> If you want to ask a question, we assume that you have read the available [Documentation](README.md).

Before you ask a question, it is best to search for existing [Issues](https://github.com/pranshuparmar/witr/issues) that might help you. In case you have found a suitable issue and still need clarification, you can write your question in this issue. It is also advisable to search the internet for answers first.

If you then still feel the need to ask a question and need clarification, we recommend the following:

- Open an [Issue](https://github.com/pranshuparmar/witr/issues/new).
- Provide as much context as you can about what you're running into.
- Provide project and platform versions (witr version, OS, Shell, target service (if possible) etc.), depending on what seems relevant.

We will then answer as soon as possible.

## I Want To Contribute

> ### Legal Notice
> When contributing to this project, you must agree that you have authored 100% of the content, that you have the necessary rights to the content and that the content you contribute may be provided under the project license.

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion for **witr**, including completely new features and minor improvements to existing functionality. Following these steps helps maintainers and the community understand your suggestion and find related suggestions.

#### Before Submitting an Enhancement

- Make sure that you are using the latest version.
- Read the [documentation](README.md) carefully and find out if the functionality is already covered, maybe by an individual configuration.
- Search [Issues](https://github.com/pranshuparmar/witr/issues) to see if the enhancement has already been suggested. If it has, add a comment to the existing issue instead of opening a new one.
- Find out whether your idea fits with the scope and aims of the project. It's up to you to make a strong case to convince the project's developers of the merits of this feature. Keep in mind that we want features that will be useful to the majority of our users and not just a small subset. If you're just targeting a edge case, might be better to create a separate extension/library.

#### How Do I Submit a Good Enhancement Suggestion?

Enhancement suggestions are tracked as [GitHub issues](https://github.com/pranshuparmar/witr/issues).

- Open an [Issue](https://github.com/pranshuparmar/witr/issues/new).
- Use a **clear and descriptive title** for the issue to identify the suggestion.
- Provide a **step-by-step description of the suggested enhancement** in as many details as possible.
- **Describe the current behavior** and explain which behavior you expected to see instead and why. At this point you can also tell which alternatives do not work for you.
- **Explain why this enhancement would be useful** to most **witr** users. You may also want to point out the other projects that solved it better and which could serve as inspiration.

### Your First Code Contribution

#### Setup

1. Fork the repository.
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/witr.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Install dependencies: `go mod download`

#### Development

- Follow the existing code style.
- Use `gofmt` to format your code.
- Write unit tests for new functionality.
- Ensure all tests pass: `go test ./...`

#### Pull Request Process

1. **Squash your commits**: We prefer a clean history. Please squash your commits into a single logical commit before submitting.
2. **Rebase on `staging`**: Ensure your branch is up to date with the `staging` branch.
3. **Open a PR**: Open a PR against the `staging` branch, not `main`.
4. **Use the Template**: Fill out the PR template completely.
5. **Review**: Wait for a maintainer to review your PR. Address any feedback promptly.
6. **Merge**: Once approved, a maintainer will merge your PR. We **strictly use Squash and Merge** to keep the `main` history clean.

### Improving Documentation

Documentation improvements are always welcome! If you find a typo or want to clarify a section, feel free to open a PR.

## Styleguides

### Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.

Format: `<type>(<scope>): <description>`

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, etc)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to our CI configuration files and scripts
- `chore`: Other changes that don't modify src or test files

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
