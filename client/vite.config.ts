import { reactRouter } from '@react-router/dev/vite'
import { defineConfig } from 'vite'

export default defineConfig({
	plugins: [reactRouter()],
	// The resort pages read the catalog from server/internal/seed, which sits
	// outside the client folder.
	server: { fs: { allow: ['..'] } },
})
