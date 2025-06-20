# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project context
This is a web app that allows users to select mountain resorts which they would like to be notified for when it snows. There are three main components, the frontend web app, the backend api layer, and a cronjob that will periodically check the weather in all resorts and fire alerts to users

## Build/Test Commands
- Build: `make build`
- Run dev environment: `make dev`
- Install dependencies: `make install`
- Server tests: `cd server && go test ./...`

## Code Style Guidelines
- **Frontend (TypeScript/React)**:
  - Use tabs for indentation, no semicolons
  - Single quotes for strings
  - Explicit type annotations for function parameters/returns
  - Use React hooks for state management
  - Material UI for components
  - Use bun as a package manager

- **Backend (Go)**:
  - Standard Go code formatting (gofmt)
  - Explicit error handling with proper returns
  - Structured logging with context
  - Use http.Handler interface pattern
  - Security headers in all responses
  - Use sqlc for all sql items
  - Use goose for all db migrations