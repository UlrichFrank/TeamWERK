/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          black: '#000000',
          white: '#FFFFFF',
          yellow: '#FAE806',
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
