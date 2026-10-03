import { createTheme } from '@mui/material/styles'

const ink = '#0b1220'
const slate = '#5b6474'
const line = '#e6e9ef'
const lineStrong = '#d3d9e2'
const glacier = '#1d6fe0'

// Forecast figures should line up like a weather readout, so numeric
// values opt into fixed-width digits.
export const tabularNums = {
	fontVariantNumeric: 'tabular-nums',
	fontFeatureSettings: '"tnum"',
} as const

const base = createTheme()

const theme = createTheme({
	palette: {
		primary: {
			main: glacier,
			dark: '#1557b8',
			light: '#e8f1fd',
			contrastText: '#ffffff',
		},
		secondary: {
			main: slate,
			light: '#eef1f5',
			dark: ink,
		},
		background: {
			default: '#fbfcfd',
			paper: '#ffffff',
		},
		text: {
			primary: ink,
			secondary: slate,
		},
		divider: line,
	},
	typography: {
		fontFamily: [
			'"Inter Variable"',
			'Inter',
			'-apple-system',
			'BlinkMacSystemFont',
			'"Segoe UI"',
			'Roboto',
			'"Helvetica Neue"',
			'Arial',
			'sans-serif',
		].join(','),
		h1: {
			fontSize: '2.375rem',
			fontWeight: 700,
			lineHeight: 1.08,
			letterSpacing: '-0.035em',
			[base.breakpoints.up('md')]: {
				fontSize: '3.25rem',
			},
		},
		h2: {
			fontSize: '1.75rem',
			fontWeight: 650,
			lineHeight: 1.2,
			letterSpacing: '-0.025em',
		},
		h3: {
			fontSize: '1.625rem',
			fontWeight: 650,
			lineHeight: 1.2,
			letterSpacing: '-0.025em',
			[base.breakpoints.up('sm')]: {
				fontSize: '1.875rem',
			},
		},
		h6: {
			fontSize: '1rem',
			fontWeight: 600,
			lineHeight: 1.4,
			letterSpacing: '-0.01em',
		},
		subtitle1: {
			fontSize: '1.0625rem',
			lineHeight: 1.6,
		},
		body1: {
			fontSize: '1rem',
			lineHeight: 1.6,
		},
		body2: {
			fontSize: '0.875rem',
			lineHeight: 1.55,
		},
		overline: {
			fontSize: '0.75rem',
			fontWeight: 600,
			lineHeight: 1.5,
			letterSpacing: '0.08em',
		},
		button: {
			textTransform: 'none',
			fontWeight: 500,
			letterSpacing: 0,
		},
	},
	shape: {
		borderRadius: 10,
	},
	components: {
		MuiCssBaseline: {
			styleOverrides: {
				body: {
					backgroundColor: '#fbfcfd',
				},
				'::selection': {
					backgroundColor: '#cfe1fb',
				},
			},
		},
		MuiButton: {
			defaultProps: {
				disableElevation: true,
			},
			styleOverrides: {
				root: {
					borderRadius: 8,
				},
				sizeLarge: {
					padding: '10px 20px',
					fontSize: '0.9375rem',
				},
				outlined: {
					borderColor: lineStrong,
					color: ink,
					'&:hover': {
						borderColor: '#b9c2ce',
						backgroundColor: '#f5f7fa',
					},
				},
			},
		},
		MuiPaper: {
			defaultProps: {
				elevation: 0,
			},
			styleOverrides: {
				root: {
					backgroundImage: 'none',
				},
				outlined: {
					borderColor: line,
					borderRadius: 14,
				},
			},
		},
		MuiMenu: {
			styleOverrides: {
				paper: {
					marginTop: 4,
					border: `1px solid ${line}`,
					boxShadow: '0 8px 24px -8px rgb(11 18 32 / 0.16)',
				},
			},
		},
		MuiDialog: {
			styleOverrides: {
				paper: {
					borderRadius: 14,
					boxShadow: '0 24px 48px -12px rgb(11 18 32 / 0.25)',
				},
			},
		},
		MuiOutlinedInput: {
			styleOverrides: {
				root: {
					borderRadius: 8,
					backgroundColor: '#ffffff',
					'&:hover .MuiOutlinedInput-notchedOutline': {
						borderColor: '#b9c2ce',
					},
				},
				notchedOutline: {
					borderColor: lineStrong,
				},
			},
		},
		MuiChip: {
			styleOverrides: {
				root: {
					borderRadius: 6,
					fontWeight: 500,
				},
			},
		},
		MuiSlider: {
			styleOverrides: {
				root: {
					height: 4,
				},
				rail: {
					backgroundColor: line,
					opacity: 1,
				},
				thumb: {
					width: 18,
					height: 18,
					backgroundColor: '#ffffff',
					border: `2px solid ${glacier}`,
					'&:hover, &.Mui-focusVisible': {
						boxShadow: '0 0 0 6px rgb(29 111 224 / 0.14)',
					},
				},
				mark: {
					display: 'none',
				},
				valueLabel: {
					backgroundColor: ink,
					borderRadius: 6,
					fontSize: '0.75rem',
				},
			},
		},
		MuiAlert: {
			styleOverrides: {
				root: {
					borderRadius: 8,
				},
			},
		},
	},
})

export default theme
