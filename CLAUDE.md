# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test Commands
- Build: `make build`
- Run dev environment: `make dev`
- Install dependencies: `make install`
- Server tests: `cd server && go test ./...`
- Client linting: `cd client && deno lint`
- Client formatting: `cd client && deno fmt`

## Code Style Guidelines
- **Frontend (TypeScript/React)**:
  - Use tabs for indentation, no semicolons
  - Single quotes for strings
  - Explicit type annotations for function parameters/returns
  - Use React hooks for state management
  - Material UI for components

- **Backend (Go)**:
  - Standard Go code formatting (gofmt)
  - Explicit error handling with proper returns
  - Structured logging with context
  - Use http.Handler interface pattern
  - Security headers in all responses