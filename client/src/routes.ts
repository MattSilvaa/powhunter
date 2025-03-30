import { index, route, type RouteConfig } from '@react-router/dev/routes'

export default [
	index('./routes/home.tsx'),
	route('/signup', './routes/signup.tsx'),
] satisfies RouteConfig
