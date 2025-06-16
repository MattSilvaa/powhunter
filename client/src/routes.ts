import { index, route, type RouteConfig } from '@react-router/dev/routes'

export default [
	index('./routes/home.tsx'),
	route('/signup', './routes/signup.tsx'),
	route('/success', './routes/success.tsx'),
] satisfies RouteConfig
