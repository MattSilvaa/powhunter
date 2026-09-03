import { afterEach } from 'bun:test'
import { GlobalRegistrator } from '@happy-dom/global-registrator'

// Register happy-dom globally before any tests run
GlobalRegistrator.register()

// Imported after registration: Testing Library reaches for document at import
// time, so it needs the DOM globals to already exist.
const { cleanup } = await import('@testing-library/react')

// Testing Library only auto-registers this when it can see a global afterEach
// as it loads, which is not something bun guarantees — it held on one version
// and not another, so the suite passed locally and failed in CI. Without
// cleanup, every render stays in document.body, and the second render of a
// component makes a getByText query match twice and throw.
afterEach(cleanup)
