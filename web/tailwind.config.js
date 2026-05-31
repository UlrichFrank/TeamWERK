/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          black: '#181310',
          white: '#FFFFFF',
          yellow: '#FDE400',
          gray: '#E5E7EB',
          blue: '#3E4A98',
          'blue-dark': '#2e3a7a',
          green: '#6EB42E',
          'green-dark': '#5a9324',
          error: '#EF4444',
          'error-light': '#FEE2E2',
          success: '#10B981',
          'success-light': '#D1FAE5',
          warning: '#F59E0B',
          'warning-light': '#FEF3C7',
          info: '#3B82F6',
          'info-light': '#DBEAFE',
          // semantic tokens
          'surface-card': '#F9FAFB',
          'text': '#111827',
          'text-muted': '#6B7280',
          'text-subtle': '#9CA3AF',
          'border': '#D1D5DB',
          'border-subtle': '#E5E7EB',
          'table-select': '#E5E7EB',
          'danger': '#C0253A',
          'danger-light': '#FCEEF1',
        },
      },
      fontFamily: {
        sans: ['"Hanken Grotesk"', 'sans-serif'],
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
