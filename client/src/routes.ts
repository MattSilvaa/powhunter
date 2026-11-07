import { index, route, type RouteConfig } from '@react-router/dev/routes'

export default [
	index('./routes/home.tsx'),
	route('/signup', './routes/signup.tsx'),
	route('/success', './routes/success.tsx'),
	route('/manage', './routes/manage.tsx'),
	route('/resorts/:resortUuid/history', './routes/resortHistory.tsx'),
	route('/login', './routes/login.tsx'),
	route('/contact', './routes/contactUs.tsx'),
	route('/resorts', './routes/resorts.tsx'),
	route('/resorts/:slug', './routes/resort.tsx'),
	route('/alert/:code', './routes/alertLink.tsx'),
	route('*', './routes/notFound.tsx'),
] satisfies RouteConfig
